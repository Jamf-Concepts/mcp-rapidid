// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
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

// connectJobCreateBody is the body for a new job: no id and no version.
type connectJobCreateBody struct {
	Name        string           `json:"name"`
	Project     string           `json:"project"`
	Description string           `json:"description,omitempty"`
	CronSpec    string           `json:"cronSpec,omitempty"`
	TimeZone    string           `json:"timeZone,omitempty"`
	Disabled    bool             `json:"disabled"`
	Action      connectJobAction `json:"action"`
}

// CreateConnectJob creates a scheduled job. It refuses a name that matches
// an existing job's ignoring case (the server only refuses exact matches),
// checks that the action set exists in the job's project, and creates the
// job disabled unless the caller says otherwise. See ConnectJobSave.go.
func CreateConnectJob(ctx context.Context, req *mcp.CallToolRequest, input CreateConnectJobInput) (*mcp.CallToolResult, rapididentity.ConnectJob, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, createConnectJobToolName)
	if setupErr != nil {
		return nil, rapididentity.ConnectJob{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, createConnectJobToolName, start, &err)

	sh.Logger().Info(createConnectJobToolName+" tool called", "name", input.Name, "project", input.Project)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	name := strings.TrimSpace(input.Name)
	project := strings.TrimSpace(input.Project)
	actionSetName := strings.TrimSpace(input.ActionSet)
	if name == "" || project == "" || actionSetName == "" {
		err = fmt.Errorf("a job name, a project and an action set are required")
		LogRIError(ctx, sh, "unable to create Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

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

	body, err := json.Marshal(connectJobCreateBody{
		Name:        name,
		Project:     actionSet.Project,
		Description: input.Description,
		CronSpec:    input.CronSpec,
		TimeZone:    input.TimeZone,
		Disabled:    disabled,
		Action:      connectJobAction{Name: actionSet.Name, Project: actionSet.Project, Args: args},
	})
	if err != nil {
		sh.Logger().Error("unable to marshal Connect job payload", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	job, err := postConnectJob(ctx, client, sh, createConnectJobToolName, body)
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Info("Created Connect job successfully", "id", job.Id, "disabled", job.Disabled)

	return nil, job, nil
}
