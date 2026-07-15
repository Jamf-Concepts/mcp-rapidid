// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const deleteConnectActionToolName = "delete-connect-action"

func DeleteConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.DeleteConnectActionByIdInput) (*mcp.CallToolResult, rapididentity.DeleteConnectActionByIdOutput, error) {
	client, th, err := ToolSetup(req, deleteConnectActionToolName)
	if err != nil {
		return nil, rapididentity.DeleteConnectActionByIdOutput{}, err
	}

	th.Logger().Info(deleteConnectActionToolName+" tool called", "id", input.Id)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Deleting Connect action by ID")
	th.Notify().Info("Deleting Connect action")
	result, err := client.DeleteConnectActionById(ctx, input)
	if err != nil {
		LogRIError(th, "unable to delete Connect action", err)
		return nil, rapididentity.DeleteConnectActionByIdOutput{}, err
	}

	th.Logger().Debug("Delete Connect action response", "result", result)
	th.Logger().Info("Deleted Connect action successfully")
	th.Notify().Info("Deleted Connect action successfully")

	return nil, *result, nil
}
