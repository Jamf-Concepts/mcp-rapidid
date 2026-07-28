// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const saveConnectActionToolName = "save-connect-action"

func SaveConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.SaveConnectActionInput) (*mcp.CallToolResult, rapididentity.SaveConnectActionOutput, error) {
	var err error
	ctx, client, th, setupErr := ToolSetup(ctx, req, saveConnectActionToolName)
	if setupErr != nil {
		return nil, rapididentity.SaveConnectActionOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, saveConnectActionToolName, start, &err)

	th.Logger().Info(saveConnectActionToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Saving Connect action")
	th.Notify().Info("Saving Connect action")
	result, err := client.SaveConnectAction(ctx, input)
	if err != nil {
		LogRIError(ctx, th, "unable to save Connect action", err)
		return nil, rapididentity.SaveConnectActionOutput{}, err
	}

	th.Logger().Debug("Save Connect action response", "result", result)
	th.Logger().Info("Saved Connect action successfully")
	th.Notify().Info("Saved Connect action successfully")

	return nil, *result, nil
}
