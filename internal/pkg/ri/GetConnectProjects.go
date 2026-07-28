// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectProjectToolName = "get-connect-projects"

type GetConnectProjectsInput struct{}

func GetConnectProjects(ctx context.Context, req *mcp.CallToolRequest, input GetConnectProjectsInput) (*mcp.CallToolResult, rapididentity.GetConnectProjectsOutput, error) {
	var err error
	ctx, client, th, setupErr := ToolSetup(ctx, req, getConnectProjectToolName)
	if setupErr != nil {
		return nil, rapididentity.GetConnectProjectsOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectProjectToolName, start, &err)

	th.Logger().Info(getConnectProjectToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if cerr := c.Close(); cerr != nil {
			LogRIError(ctx, th, "unable to close connection to rapididentity", cerr)
		}
	}(client)

	th.Logger().Info("Call Connect projects endpoint")
	th.Notify().Info("Calling RapidIdentity Connect projects endpoint")
	result, err := client.GetConnectProjects(ctx)
	if err != nil {
		LogRIError(ctx, th, "unable to retrieve rapididentity connect projects", err)
		return nil, rapididentity.GetConnectProjectsOutput{}, err
	}

	th.Logger().Debug("Response payload", "projects", result)

	th.Logger().Info("Retrieved Connect projects successfully", "projectTotal", len(result.Projects))
	th.Notify().Info(fmt.Sprintf("Retrieved %d projects", len(result.Projects)))

	return nil, *result, nil
}
