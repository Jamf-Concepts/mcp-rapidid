// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const createConnectJobToolName = "create-connect-job"

type CreateConnectJobInput struct {
	Name        string               `json:"name" jsonschema:"The name of the new job. It must not match an existing job's name, ignoring case"`
	Project     string               `json:"project" jsonschema:"The project the job belongs to. The action set must be in this project"`
	ActionSet   string               `json:"actionSet" jsonschema:"The name of the action set the job runs"`
	Args        []ConnectJobArgInput `json:"args,omitempty" jsonschema:"Values for the action set's arguments. Arguments left out have no value"`
	CronSpec    string               `json:"cronSpec,omitempty" jsonschema:"The schedule in Quartz cron format with seconds: <second> <minute> <hour> <day of month> <month> <day of week>. For example every 5 minutes is 0 */5 * * * ?"`
	TimeZone    string               `json:"timeZone,omitempty" jsonschema:"The time zone for the schedule, for example America/Chicago. The server uses UTC when it is left out"`
	Description string               `json:"description,omitempty" jsonschema:"The description of the job"`
	Disabled    *bool                `json:"disabled,omitempty" jsonschema:"Whether the job is disabled. New jobs are disabled unless this is false"`
}

// createConnectJobBody is the body for a new job. It has no id or version,
// so that the server creates the job, and its action has no id, so that the
// server generates the job's action step id.
type createConnectJobBody struct {
	Name        string                 `json:"name"`
	Project     string                 `json:"project"`
	Description string                 `json:"description,omitempty"`
	CronSpec    string                 `json:"cronSpec,omitempty"`
	TimeZone    string                 `json:"timeZone,omitempty"`
	Disabled    bool                   `json:"disabled"`
	Action      createConnectJobAction `json:"action"`
}

type createConnectJobAction struct {
	Name    string                   `json:"name"`
	Project string                   `json:"project"`
	Args    rapididentity.ArgDefList `json:"args"`
}

// CreateConnectJob creates a scheduled job. It refuses a name that matches
// an existing job's ignoring case (the server only refuses exact matches),
// checks that the action set exists in the job's project, and creates the
// job disabled unless the caller says otherwise. See ConnectJobs.go.
func CreateConnectJob(ctx context.Context, req *mcp.CallToolRequest, input CreateConnectJobInput) (*mcp.CallToolResult, rapididentity.ConnectJob, error) {
	name := strings.TrimSpace(input.Name)
	project := strings.TrimSpace(input.Project)
	actionSetName := strings.TrimSpace(input.ActionSet)
	if name == "" || project == "" || actionSetName == "" {
		return nil, rapididentity.ConnectJob{}, fmt.Errorf("a job name, a project and an action set are required")
	}

	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, createConnectJobToolName)
	if setupErr != nil {
		return nil, rapididentity.ConnectJob{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, createConnectJobToolName, start, &err)

	sh.Logger().Info(createConnectJobToolName+" tool called", "name", name, "project", project)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	jobs, err := client.GetConnectJobs(ctx, rapididentity.GetConnectJobsInput{})
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve rapididentity connect jobs", err)
		return nil, rapididentity.ConnectJob{}, err
	}
	for _, job := range jobs.Jobs {
		if strings.EqualFold(job.Name, name) {
			err = fmt.Errorf("a Connect job named %q already exists", job.Name)
			LogRIError(ctx, sh, "unable to create Connect job", err)
			return nil, rapididentity.ConnectJob{}, err
		}
	}

	actionSet, err := lookupConnectActionSet(ctx, client, sh, project, actionSetName)
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}
	args, err := applyConnectJobArgs(actionSet, connectJobArgsFor(actionSet, nil), input.Args)
	if err != nil {
		LogRIError(ctx, sh, "unable to create Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	disabled := true
	if input.Disabled != nil {
		disabled = *input.Disabled
	}

	requestPayload, err := json.Marshal(createConnectJobBody{
		Name:        name,
		Project:     actionSet.Project,
		Description: input.Description,
		CronSpec:    input.CronSpec,
		TimeZone:    input.TimeZone,
		Disabled:    disabled,
		Action:      createConnectJobAction{Name: actionSet.Name, Project: actionSet.Project, Args: args},
	})
	if err != nil {
		sh.Logger().Error("unable to marshal Connect job payload", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Info("Calling admin/connect/jobs endpoint")
	jobRes, err := client.DoCustomRequest(ctx, "POST", "admin/connect/jobs", bytes.NewBuffer(requestPayload))
	if err != nil {
		LogRIError(ctx, sh, "unable to create Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for admin/connect/jobs endpoint response", "error", err)
		}
	}(jobRes)

	if err = checkResponseStatus(jobRes); err != nil {
		LogRIError(ctx, sh, "unable to create Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	jobBody, err := io.ReadAll(jobRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for admin/connect/jobs response", "error", err, "status", jobRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Debug("POST admin/connect/jobs response body", "body", string(jobBody))

	var job rapididentity.ConnectJob

	err = json.Unmarshal(jobBody, &job)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for POST admin/connect/jobs response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Info("Created Connect job successfully", "id", job.Id, "disabled", job.Disabled)

	return nil, job, nil
}
