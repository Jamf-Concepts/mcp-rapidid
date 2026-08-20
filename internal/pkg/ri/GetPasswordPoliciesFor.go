// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getPasswordPoliciesForToolName = "get-password-policies-for"

func GetPasswordPoliciesFor(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetPasswordPoliciesForInput) (*mcp.CallToolResult, rapididentity.PasswordPolicy, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getPasswordPoliciesForToolName)
	if setupErr != nil {
		return nil, rapididentity.PasswordPolicy{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getPasswordPoliciesForToolName, start, &err)

	sh.Logger().Info(getPasswordPoliciesForToolName+" tool called", "userCount", len(input.UserIds))

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	sh.Logger().Info("Getting password policies for user")
	result, err := client.GetPasswordPoliciesFor(ctx, input)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve password policies", err)
		return nil, rapididentity.PasswordPolicy{}, err
	}

	sh.Logger().Debug("Get password policies response", "result", result)
	sh.Logger().Info("Retrieved password policies successfully")

	return nil, *result, nil
}
