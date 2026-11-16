// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const deleteConnectProjectToolName = "delete-connect-project"

type DeleteConnectProjectInput struct {
	Name string `json:"name" jsonschema:"The name of the Connect project to delete"`
}

type DeleteConnectProjectOutput struct {
	DeleteOperationStatus rapididentity.OperationStatus `json:"deleteOperationStatus" jsonschema:"The result of the Connect project delete operation."`
}

// DeleteConnectProject deletes a project by name. It looks the project up
// first and deletes it by its exact name, which is the project's key on the
// server, rather than by id: after a project has been replaced, its old id
// can still answer requests. A response that says success is false is
// returned as an error, so the project is never reported deleted when it
// was not.
func DeleteConnectProject(ctx context.Context, req *mcp.CallToolRequest, input DeleteConnectProjectInput) (*mcp.CallToolResult, DeleteConnectProjectOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, deleteConnectProjectToolName)
	if setupErr != nil {
		return nil, DeleteConnectProjectOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, deleteConnectProjectToolName, start, &err)

	sh.Logger().Info(deleteConnectProjectToolName+" tool called", "name", input.Name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	name := strings.TrimSpace(input.Name)
	if name == "" {
		err = fmt.Errorf("a project name is required")
		LogRIError(ctx, sh, "unable to delete Connect project", err)
		return nil, DeleteConnectProjectOutput{}, err
	}

	existingName, err := findConnectProjectName(ctx, client, sh, name)
	if err != nil {
		return nil, DeleteConnectProjectOutput{}, err
	}
	if existingName == "" {
		err = fmt.Errorf("no Connect project named %q exists", name)
		LogRIError(ctx, sh, "unable to delete Connect project", err)
		return nil, DeleteConnectProjectOutput{}, err
	}

	path := "admin/connect/projects/" + url.PathEscape(existingName)

	sh.Logger().Info("Deleting Connect project", "path", path)
	deleteRes, err := client.DoCustomRequest(ctx, "DELETE", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to delete Connect project", err)
		return nil, DeleteConnectProjectOutput{}, err
	}

	deleteBody, err := readConnectProjectResponse(ctx, sh, deleteConnectProjectToolName, deleteRes)
	if err != nil {
		return nil, DeleteConnectProjectOutput{}, err
	}

	sh.Logger().Debug("DELETE "+path+" response body", "body", string(deleteBody))

	var status rapididentity.OperationStatus

	err = json.Unmarshal(deleteBody, &status)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for DELETE "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", deleteConnectProjectToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, DeleteConnectProjectOutput{}, err
	}

	if !status.Success {
		err = fmt.Errorf("Connect did not delete project %q: %s", existingName, status.Message)
		LogRIError(ctx, sh, "unable to delete Connect project", err)
		return nil, DeleteConnectProjectOutput{}, err
	}

	sh.Logger().Info("Deleted Connect project successfully", "name", existingName)

	return nil, DeleteConnectProjectOutput{DeleteOperationStatus: status}, nil
}
