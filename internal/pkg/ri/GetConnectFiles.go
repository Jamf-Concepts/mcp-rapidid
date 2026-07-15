// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectFilesToolName = "get-connect-files"

func GetConnectFiles(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectFilesInput) (*mcp.CallToolResult, rapididentity.GetConnectFilesOutput, error) {
	client, th, err := ToolSetup(req, getConnectFilesToolName)
	if err != nil {
		return nil, rapididentity.GetConnectFilesOutput{}, err
	}

	th.Logger().Info(getConnectFilesToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Getting Connect files")
	th.Notify().Info("Retrieving Connect files")
	result, err := client.GetConnectFiles(ctx, input)
	if err != nil {
		LogRIError(th, "unable to retrieve Connect files", err)
		return nil, rapididentity.GetConnectFilesOutput{}, err
	}

	th.Logger().Debug("Get Connect files response", "result", result)
	th.Logger().Info("Retrieved Connect files successfully")
	th.Notify().Info("Retrieved Connect files successfully")

	return nil, *result, nil
}
