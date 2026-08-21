// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const deleteConnectActionToolName = "delete-connect-action"

func DeleteConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.DeleteConnectActionByIdInput) (*mcp.CallToolResult, rapididentity.DeleteConnectActionByIdOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, deleteConnectActionToolName)
	if setupErr != nil {
		return nil, rapididentity.DeleteConnectActionByIdOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, deleteConnectActionToolName, start, &err)

	sh.Logger().Info(deleteConnectActionToolName+" tool called", "id", input.Id)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	sh.Logger().Info("Deleting Connect action by ID")
	result, err := client.DeleteConnectActionById(ctx, input)
	if err != nil {
		LogRIError(ctx, sh, "unable to delete Connect action", err)
		return nil, rapididentity.DeleteConnectActionByIdOutput{}, err
	}

	sh.Logger().Debug("Delete Connect action response", "result", result)
	sh.Logger().Info("Deleted Connect action successfully")

	return nil, *result, nil
}
