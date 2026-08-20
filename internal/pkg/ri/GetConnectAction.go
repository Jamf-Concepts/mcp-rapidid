// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectActionToolName = "get-connect-action"

func GetConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectActionByIdInput) (*mcp.CallToolResult, rapididentity.GetConnectActionByIdOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getConnectActionToolName)
	if setupErr != nil {
		return nil, rapididentity.GetConnectActionByIdOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectActionToolName, start, &err)

	sh.Logger().Info(getConnectActionToolName+" tool called", "id", input.Id)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	sh.Logger().Info("Getting Connect action by ID")
	result, err := client.GetConnectActionById(ctx, input)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect action", err)
		return nil, rapididentity.GetConnectActionByIdOutput{}, err
	}

	sh.Logger().Debug("Get Connect action response", "result", result)
	sh.Logger().Info("Retrieved Connect action successfully")

	return nil, *result, nil
}
