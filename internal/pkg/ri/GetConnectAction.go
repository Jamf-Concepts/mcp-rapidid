// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectActionToolName = "get-connect-action"

func GetConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectActionByIdInput) (*mcp.CallToolResult, rapididentity.GetConnectActionByIdOutput, error) {
	client, th, err := ToolSetup(req, getConnectActionToolName)
	if err != nil {
		return nil, rapididentity.GetConnectActionByIdOutput{}, err
	}

	th.Logger().Info(getConnectActionToolName+" tool called", "id", input.Id)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Getting Connect action by ID")
	th.Notify().Info("Retrieving Connect action")
	result, err := client.GetConnectActionById(ctx, input)
	if err != nil {
		LogRIError(th, "unable to retrieve Connect action", err)
		return nil, rapididentity.GetConnectActionByIdOutput{}, err
	}

	th.Logger().Debug("Get Connect action response", "result", result)
	th.Logger().Info("Retrieved Connect action successfully")
	th.Notify().Info("Retrieved Connect action successfully")

	return nil, *result, nil
}
