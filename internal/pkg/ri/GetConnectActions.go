// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectActionsToolName = "get-connect-actions"

func GetConnectActions(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectActionsInput) (*mcp.CallToolResult, rapididentity.GetConnectActionsOutput, error) {
	var err error
	ctx, client, th, setupErr := ToolSetup(ctx, req, getConnectActionsToolName)
	if setupErr != nil {
		return nil, rapididentity.GetConnectActionsOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectActionsToolName, start, &err)

	th.Logger().Info(getConnectActionsToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Getting Connect actions")
	th.Notify().Info("Retrieving Connect actions")
	result, err := client.GetConnectActions(ctx, input)
	if err != nil {
		LogRIError(ctx, th, "unable to retrieve Connect actions", err)
		return nil, rapididentity.GetConnectActionsOutput{}, err
	}

	th.Logger().Debug("Get Connect actions response", "result", result)
	th.Logger().Info("Retrieved Connect actions successfully")
	th.Notify().Info("Retrieved Connect actions successfully")

	return nil, *result, nil
}
