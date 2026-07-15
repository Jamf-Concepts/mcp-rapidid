// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const runConnectActionToolName = "run-connect-action"

func RunConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.RunConnectActionInput) (*mcp.CallToolResult, rapididentity.RunConnectActionOutput, error) {
	client, th, err := ToolSetup(req, runConnectActionToolName)
	if err != nil {
		return nil, rapididentity.RunConnectActionOutput{}, err
	}

	th.Logger().Info(runConnectActionToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Running Connect action")
	th.Notify().Info("Running Connect action")
	result, err := client.RunConnectAction(ctx, input)
	if err != nil {
		LogRIError(th, "unable to run Connect action", err)
		return nil, rapididentity.RunConnectActionOutput{}, err
	}

	th.Logger().Debug("Run Connect action response", "result", result)
	th.Logger().Info("Connect action ran successfully")
	th.Notify().Info("Connect action ran successfully")

	return nil, *result, nil
}
