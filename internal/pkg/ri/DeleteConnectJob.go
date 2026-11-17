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

const deleteConnectJobToolName = "delete-connect-job"

type DeleteConnectJobInput struct {
	Name string `json:"name" jsonschema:"The name of the Connect job to delete. Job names are unique across all projects"`
}

type DeleteConnectJobOutput struct {
	DeleteOperationStatus rapididentity.OperationStatus `json:"deleteOperationStatus" jsonschema:"The result of the Connect job delete operation."`
}

// DeleteConnectJob deletes a job, looked up by name and addressed by its id
// (see requireConnectJob). It deletes only the job, not the action set the
// job runs. A response that says success is false is returned as an error,
// so a job is never reported deleted when it was not.
func DeleteConnectJob(ctx context.Context, req *mcp.CallToolRequest, input DeleteConnectJobInput) (*mcp.CallToolResult, DeleteConnectJobOutput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, DeleteConnectJobOutput{}, fmt.Errorf("a job name is required")
	}

	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, deleteConnectJobToolName)
	if setupErr != nil {
		return nil, DeleteConnectJobOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, deleteConnectJobToolName, start, &err)

	sh.Logger().Info(deleteConnectJobToolName+" tool called", "name", name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	existing, err := requireConnectJob(ctx, client, sh, name)
	if err != nil {
		return nil, DeleteConnectJobOutput{}, err
	}

	path := "admin/connect/jobs/" + url.PathEscape(existing.Id)

	sh.Logger().Info("Deleting Connect job", "path", path)
	deleteRes, err := client.DoCustomRequest(ctx, "DELETE", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to delete Connect job", err)
		return nil, DeleteConnectJobOutput{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(deleteRes)

	if err = checkResponseStatus(deleteRes); err != nil {
		LogRIError(ctx, sh, "unable to delete Connect job", err)
		return nil, DeleteConnectJobOutput{}, err
	}

	deleteBody, err := io.ReadAll(deleteRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for DELETE "+path+" response", "error", err, "status", deleteRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", deleteConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, DeleteConnectJobOutput{}, err
	}

	sh.Logger().Debug("DELETE "+path+" response body", "body", string(deleteBody))

	var status rapididentity.OperationStatus

	err = json.Unmarshal(deleteBody, &status)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for DELETE "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", deleteConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, DeleteConnectJobOutput{}, err
	}

	if !status.Success {
		err = fmt.Errorf("Connect did not delete job %q: %s", existing.Name, status.Message)
		LogRIError(ctx, sh, "unable to delete Connect job", err)
		return nil, DeleteConnectJobOutput{}, err
	}

	sh.Logger().Info("Deleted Connect job successfully", "name", existing.Name)

	return nil, DeleteConnectJobOutput{DeleteOperationStatus: status}, nil
}
