// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectJobsToolName = "get-connect-jobs"

func GetConnectJobs(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectJobsInput) (*mcp.CallToolResult, rapididentity.GetConnectJobsOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getConnectJobsToolName)
	if setupErr != nil {
		return nil, rapididentity.GetConnectJobsOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectJobsToolName, start, &err)

	sh.Logger().Info(getConnectJobsToolName+" tool called", "project", input.Project)

	defer func(c *rapididentity.Client) {
		if cerr := c.Close(); cerr != nil {
			LogRIError(ctx, sh, "unable to close connection to rapididentity", cerr)
		}
	}(client)

	sh.Logger().Info("Call Connect jobs endpoint")
	result, err := client.GetConnectJobs(ctx, input)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve rapididentity connect jobs", err)
		return nil, rapididentity.GetConnectJobsOutput{}, err
	}

	sh.Logger().Debug("Response payload", "jobs", result)

	sh.Logger().Info("Retrieved Connect jobs successfully", "jobTotal", len(result.Jobs))

	return nil, *result, nil
}
