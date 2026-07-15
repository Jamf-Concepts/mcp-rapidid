// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectActionsToolName = "get-connect-actions"

func GetConnectActions(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectActionsInput) (*mcp.CallToolResult, rapididentity.GetConnectActionsOutput, error) {
	client, th, err := ToolSetup(req, getConnectActionsToolName)
	if err != nil {
		return nil, rapididentity.GetConnectActionsOutput{}, err
	}

	th.Logger().Info(getConnectActionsToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Getting Connect actions")
	th.Notify().Info("Retrieving Connect actions")
	result, err := client.GetConnectActions(ctx, input)
	if err != nil {
		LogRIError(th, "unable to retrieve Connect actions", err)
		return nil, rapididentity.GetConnectActionsOutput{}, err
	}

	th.Logger().Debug("Get Connect actions response", "result", result)
	th.Logger().Info("Retrieved Connect actions successfully")
	th.Notify().Info("Retrieved Connect actions successfully")

	return nil, *result, nil
}
