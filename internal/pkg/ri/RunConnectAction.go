// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const runConnectActionToolName = "run-connect-action"

func RunConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.RunConnectActionInput) (*mcp.CallToolResult, rapididentity.RunConnectActionOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, runConnectActionToolName)
	if setupErr != nil {
		return nil, rapididentity.RunConnectActionOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, runConnectActionToolName, start, &err)

	sh.Logger().Info(runConnectActionToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	sh.Logger().Info("Running Connect action")
	result, err := client.RunConnectAction(ctx, input)
	if err != nil {
		LogRIError(ctx, sh, "unable to run Connect action", err)
		return nil, rapididentity.RunConnectActionOutput{}, err
	}

	sh.Logger().Debug("Run Connect action response", "result", result)
	sh.Logger().Info("Connect action ran successfully")

	return nil, *result, nil
}
