// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const setPasswordToolName = "set-password"

type SetPasswordOutput struct {
	Result rapididentity.SetPasswordOutput `json:"result" jsonschema:"The set password result"`
}

func SetPassword(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.SetPasswordInput) (*mcp.CallToolResult, SetPasswordOutput, error) {
	client, th, err := ToolSetup(req, setPasswordToolName)
	if err != nil {
		return nil, SetPasswordOutput{}, err
	}

	th.Logger().Info(setPasswordToolName+" tool called", "delegationId", input.DelegationId)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Setting password for user")
	th.Notify().Info("Setting password")
	result, err := client.SetPassword(ctx, input)
	if err != nil {
		LogRIError(th, "unable to set password", err)
		return nil, SetPasswordOutput{}, err
	}

	th.Logger().Info("Password set successfully")
	th.Notify().Info("Password set successfully")

	return nil, SetPasswordOutput{Result: result}, nil
}
