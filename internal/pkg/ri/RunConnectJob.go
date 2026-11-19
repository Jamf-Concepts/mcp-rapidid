// Copyright 2026, Jamf Software LLC

package ri

import (
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

const runConnectJobToolName = "run-connect-job"

type RunConnectJobInput struct {
	Name string `json:"name" jsonschema:"The name of the Connect job to run. Job names are unique across all projects"`
}

type RunConnectJobOutput struct {
	RunOperationStatus rapididentity.OperationStatus `json:"runOperationStatus" jsonschema:"The result of starting the Connect job."`
}

// RunConnectJob runs a job now, exactly as it is configured. GET
// /admin/connect/runJob/{nameOrId} takes no arguments, so a job cannot be
// run in log-only mode this way; to try a job's action set without writes,
// run the action set with run-connect-action and logOnly instead. The
// endpoint returns only a status, not the job's log. A response that says
// success is false is returned as an error, so a job is never reported as
// started when it was not.
func RunConnectJob(ctx context.Context, req *mcp.CallToolRequest, input RunConnectJobInput) (*mcp.CallToolResult, RunConnectJobOutput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, RunConnectJobOutput{}, fmt.Errorf("a job name is required")
	}

	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, runConnectJobToolName)
	if setupErr != nil {
		return nil, RunConnectJobOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, runConnectJobToolName, start, &err)

	sh.Logger().Info(runConnectJobToolName+" tool called", "name", name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	existing, err := requireConnectJob(ctx, client, sh, name)
	if err != nil {
		return nil, RunConnectJobOutput{}, err
	}

	path := "admin/connect/runJob/" + url.PathEscape(existing.Id)

	sh.Logger().Info("Running Connect job", "path", path)
	runRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to run Connect job", err)
		return nil, RunConnectJobOutput{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(runRes)

	if err = checkResponseStatus(runRes); err != nil {
		LogRIError(ctx, sh, "unable to run Connect job", err)
		return nil, RunConnectJobOutput{}, err
	}

	runBody, err := io.ReadAll(runRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for GET "+path+" response", "error", err, "status", runRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", runConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, RunConnectJobOutput{}, err
	}

	sh.Logger().Debug("GET "+path+" response body", "body", string(runBody))

	var status rapididentity.OperationStatus

	err = json.Unmarshal(runBody, &status)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", runConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, RunConnectJobOutput{}, err
	}

	if !status.Success {
		err = fmt.Errorf("Connect did not run job %q: %s", existing.Name, status.Message)
		LogRIError(ctx, sh, "unable to run Connect job", err)
		return nil, RunConnectJobOutput{}, err
	}

	sh.Logger().Info("Started Connect job successfully", "name", existing.Name)

	return nil, RunConnectJobOutput{RunOperationStatus: status}, nil
}
