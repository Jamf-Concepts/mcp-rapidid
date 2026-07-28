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
	ctx, client, th, setupErr := ToolSetup(ctx, req, getPasswordPoliciesForToolName)
	if setupErr != nil {
		return nil, rapididentity.PasswordPolicy{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getPasswordPoliciesForToolName, start, &err)

	th.Logger().Info(getPasswordPoliciesForToolName+" tool called", "userCount", len(input.UserIds))

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Getting password policies for user")
	th.Notify().Info("Retrieving password policies")
	result, err := client.GetPasswordPoliciesFor(ctx, input)
	if err != nil {
		LogRIError(ctx, th, "unable to retrieve password policies", err)
		return nil, rapididentity.PasswordPolicy{}, err
	}

	th.Logger().Debug("Get password policies response", "result", result)
	th.Logger().Info("Retrieved password policies successfully")
	th.Notify().Info("Retrieved password policies successfully")

	return nil, *result, nil
}
