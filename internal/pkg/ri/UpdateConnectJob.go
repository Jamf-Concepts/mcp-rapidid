// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
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
// so the current job is read as raw JSON per field and sent back with its id
// and version; a job changed by someone else in between is refused by the
// server (409). See ConnectJobSave.go.
func UpdateConnectJob(ctx context.Context, req *mcp.CallToolRequest, input UpdateConnectJobInput) (*mcp.CallToolResult, rapididentity.ConnectJob, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, updateConnectJobToolName)
	if setupErr != nil {
		return nil, rapididentity.ConnectJob{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, updateConnectJobToolName, start, &err)

	sh.Logger().Info(updateConnectJobToolName+" tool called", "name", input.Name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	changes := connectJobChanges{
		Description: input.Description,
		CronSpec:    input.CronSpec,
		TimeZone:    input.TimeZone,
		Disabled:    input.Disabled,
	}
	changesAction := input.ActionSet != nil || len(input.Args) > 0
	if changes == (connectJobChanges{}) && !changesAction {
		err = fmt.Errorf("nothing to update: pass at least one field to change")
		LogRIError(ctx, sh, "unable to update Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	existing, err := requireConnectJob(ctx, client, sh, input.Name)
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}

	payload, err := getConnectJobJSON(ctx, client, sh, existing.Id)
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}

	if changesAction {
		var action connectJobAction
		action, err = updatedConnectJobAction(ctx, client, sh, payload, existing.Project, input)
		if err != nil {
			return nil, rapididentity.ConnectJob{}, err
		}
		if payload["action"], err = json.Marshal(action); err != nil {
			return nil, rapididentity.ConnectJob{}, err
		}
	}

	changed, err := changes.rawFields()
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}
	for field, value := range changed {
		payload[field] = value
	}

	body, err := json.Marshal(payload)
	if err != nil {
		sh.Logger().Error("unable to marshal Connect job payload", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	job, err := postConnectJob(ctx, client, sh, updateConnectJobToolName, body)
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Info("Updated Connect job successfully", "id", job.Id, "version", job.Version)

	return nil, job, nil
}

// getConnectJobJSON reads one job, by id, as raw JSON per field, so that the
// fields this tool does not change go back exactly as the server sent them.
func getConnectJobJSON(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, id string) (map[string]json.RawMessage, error) {
	path := "admin/connect/jobs/" + url.PathEscape(id)

	sh.Logger().Info("Retrieving the current Connect job", "path", path)
	res, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect job", err)
		return nil, err
	}

	body, err := readConnectJobResponse(ctx, sh, updateConnectJobToolName, res)
	if err != nil {
		return nil, err
	}

	var job map[string]json.RawMessage
	if err = json.Unmarshal(body, &job); err != nil || job == nil {
		if err == nil {
			err = fmt.Errorf("empty response retrieving Connect job %q", id)
		}
		sh.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, err
	}
	return job, nil
}

// updatedConnectJobAction returns the job's action with the requested action
// set and argument changes applied. The current action is decoded strictly:
// if it has a field this tool does not know, it is refused rather than sent
// back without that field.
func updatedConnectJobAction(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, payload map[string]json.RawMessage, project string, input UpdateConnectJobInput) (connectJobAction, error) {
	var action connectJobAction
	decoder := json.NewDecoder(bytes.NewReader(payload["action"]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&action); err != nil {
		err = fmt.Errorf("the job's action has a shape this tool does not know, so it was not changed: %w", err)
		LogRIError(ctx, sh, "unable to update Connect job", err)
		return connectJobAction{}, err
	}

	actionSetName := action.Name
	if input.ActionSet != nil {
		actionSetName = strings.TrimSpace(*input.ActionSet)
	}
	actionSet, err := lookupConnectActionSet(ctx, client, sh, project, actionSetName)
	if err != nil {
		return connectJobAction{}, err
	}

	args, err := applyConnectJobArgs(actionSet, connectJobArgsFor(actionSet, action.Args), input.Args)
	if err != nil {
		LogRIError(ctx, sh, "unable to update Connect job", err)
		return connectJobAction{}, err
	}

	action.Name = actionSet.Name
	action.Project = actionSet.Project
	action.Args = args
	return action, nil
}
