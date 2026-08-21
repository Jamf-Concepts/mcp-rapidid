// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectFilesToolName = "get-connect-files"

func GetConnectFiles(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectFilesInput) (*mcp.CallToolResult, rapididentity.GetConnectFilesOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getConnectFilesToolName)
	if setupErr != nil {
		return nil, rapididentity.GetConnectFilesOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectFilesToolName, start, &err)

	sh.Logger().Info(getConnectFilesToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	sh.Logger().Info("Getting Connect files")
	result, err := client.GetConnectFiles(ctx, input)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect files", err)
		return nil, rapididentity.GetConnectFilesOutput{}, err
	}

	sh.Logger().Debug("Get Connect files response", "result", result)
	sh.Logger().Info("Retrieved Connect files successfully")

	return nil, *result, nil
}
