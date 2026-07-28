// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const setPasswordToolName = "set-password"

type SetPasswordOutput struct {
	Result rapididentity.SetPasswordOutput `json:"result" jsonschema:"The set password result"`
}

func SetPassword(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.SetPasswordInput) (*mcp.CallToolResult, SetPasswordOutput, error) {
	var err error
	ctx, client, th, setupErr := ToolSetup(ctx, req, setPasswordToolName)
	if setupErr != nil {
		return nil, SetPasswordOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, setPasswordToolName, start, &err)

	th.Logger().Info(setPasswordToolName+" tool called", "delegationId", input.DelegationId)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Setting password for user")
	th.Notify().Info("Setting password")
	result, err := client.SetPassword(ctx, input)
	if err != nil {
		LogRIError(ctx, th, "unable to set password", err)
		return nil, SetPasswordOutput{}, err
	}

	th.Logger().Info("Password set successfully")
	th.Notify().Info("Password set successfully")

	return nil, SetPasswordOutput{Result: result}, nil
}
