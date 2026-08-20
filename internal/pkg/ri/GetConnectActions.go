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
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getConnectActionsToolName)
	if setupErr != nil {
		return nil, rapididentity.GetConnectActionsOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectActionsToolName, start, &err)

	sh.Logger().Info(getConnectActionsToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	sh.Logger().Info("Getting Connect actions")
	result, err := client.GetConnectActions(ctx, input)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect actions", err)
		return nil, rapididentity.GetConnectActionsOutput{}, err
	}

	sh.Logger().Debug("Get Connect actions response", "result", result)
	sh.Logger().Info("Retrieved Connect actions successfully")

	return nil, *result, nil
}
