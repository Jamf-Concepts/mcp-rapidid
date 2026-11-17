// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const updateConnectJobToolName = "update-connect-job"

// UpdateConnectJobInput uses pointers so that a field left out ("keep it")
// can be told apart from one being set.
type UpdateConnectJobInput struct {
	Name        string               `json:"name" jsonschema:"The name of the existing job to update. Jobs cannot be renamed or moved to another project"`
	ActionSet   *string              `json:"actionSet,omitempty" jsonschema:"The name of another action set in the job's project for the job to run. Argument values carry over for arguments the new action set also has"`
	Args        []ConnectJobArgInput `json:"args,omitempty" jsonschema:"Argument values to set, by name. Arguments left out keep their values; an empty value clears one"`
	CronSpec    *string              `json:"cronSpec,omitempty" jsonschema:"The new schedule in Quartz cron format with seconds: <second> <minute> <hour> <day of month> <month> <day of week>"`
	TimeZone    *string              `json:"timeZone,omitempty" jsonschema:"The new time zone for the schedule, for example America/Chicago"`
	Description *string              `json:"description,omitempty" jsonschema:"The new description of the job"`
	Disabled    *bool                `json:"disabled,omitempty" jsonschema:"Whether the job is disabled"`
}

// UpdateConnectJob changes the given fields of an existing job and keeps
// every other field as it is. The server replaces the whole job on update,
// so the current job is read into rapididentity.ConnectJob, changed and
// posted back with its id and version; a job changed by someone else in
// between is refused by the server (409). The read is strict: a field the
// SDK type does not have stops the update, so that it is never silently
// cleared. See ConnectJobs.go.
func UpdateConnectJob(ctx context.Context, req *mcp.CallToolRequest, input UpdateConnectJobInput) (*mcp.CallToolResult, rapididentity.ConnectJob, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, rapididentity.ConnectJob{}, fmt.Errorf("a job name is required")
	}
	changesAction := input.ActionSet != nil || len(input.Args) > 0
	if !changesAction && input.CronSpec == nil && input.TimeZone == nil && input.Description == nil && input.Disabled == nil {
		return nil, rapididentity.ConnectJob{}, fmt.Errorf("nothing to update: pass at least one field to change")
	}

	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, updateConnectJobToolName)
	if setupErr != nil {
		return nil, rapididentity.ConnectJob{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, updateConnectJobToolName, start, &err)

	sh.Logger().Info(updateConnectJobToolName+" tool called", "name", name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	existing, err := requireConnectJob(ctx, client, sh, name)
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}

	path := "admin/connect/jobs/" + url.PathEscape(existing.Id)

	sh.Logger().Info("Retrieving the current Connect job", "path", path)
	currentRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(currentRes)

	if err = checkResponseStatus(currentRes); err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	currentBody, err := io.ReadAll(currentRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for GET "+path+" response", "error", err, "status", currentRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	var job rapididentity.ConnectJob
	decoder := json.NewDecoder(bytes.NewReader(currentBody))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&job); err != nil {
		err = fmt.Errorf("Connect job %q could not be read safely, so it was not changed (a field this tool does not know would be lost): %w", existing.Name, err)
		LogRIError(ctx, sh, "unable to update Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	if changesAction {
		actionSetName := job.Action.Name
		if input.ActionSet != nil {
			actionSetName = strings.TrimSpace(*input.ActionSet)
		}
		var actionSet *rapididentity.ActionDef
		actionSet, err = lookupConnectActionSet(ctx, client, sh, job.Project, actionSetName)
		if err != nil {
			return nil, rapididentity.ConnectJob{}, err
		}
		var args rapididentity.ArgDefList
		args, err = applyConnectJobArgs(actionSet, connectJobArgsFor(actionSet, job.Action.Args), input.Args)
		if err != nil {
			LogRIError(ctx, sh, "unable to update Connect job", err)
			return nil, rapididentity.ConnectJob{}, err
		}
		job.Action.Name = actionSet.Name
		job.Action.Project = actionSet.Project
		job.Action.Args = args
	}
	if input.CronSpec != nil {
		job.CronSpec = *input.CronSpec
	}
	if input.TimeZone != nil {
		job.TimeZone = *input.TimeZone
	}
	if input.Description != nil {
		job.Description = *input.Description
	}
	if input.Disabled != nil {
		job.Disabled = *input.Disabled
	}

	requestPayload, err := json.Marshal(job)
	if err != nil {
		sh.Logger().Error("unable to marshal Connect job payload", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Info("Calling admin/connect/jobs endpoint")
	jobRes, err := client.DoCustomRequest(ctx, "POST", "admin/connect/jobs", bytes.NewBuffer(requestPayload))
	if err != nil {
		LogRIError(ctx, sh, "unable to update Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for admin/connect/jobs endpoint response", "error", err)
		}
	}(jobRes)

	if err = checkResponseStatus(jobRes); err != nil {
		LogRIError(ctx, sh, "unable to update Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	jobBody, err := io.ReadAll(jobRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for admin/connect/jobs response", "error", err, "status", jobRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Debug("POST admin/connect/jobs response body", "body", string(jobBody))

	var updated rapididentity.ConnectJob

	err = json.Unmarshal(jobBody, &updated)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for POST admin/connect/jobs response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Info("Updated Connect job successfully", "id", updated.Id, "version", updated.Version)

	return nil, updated, nil
}
