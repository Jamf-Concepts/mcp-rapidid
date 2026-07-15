// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectFileContentToolName = "get-connect-file-content"

type GetConnectFileContentOutput struct {
	Content string `json:"content" jsonschema:"The text content of the Connect file"`
}

func GetConnectFileContent(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectFileContentInput) (*mcp.CallToolResult, GetConnectFileContentOutput, error) {
	client, th, err := ToolSetup(req, getConnectFileContentToolName)
	if err != nil {
		return nil, GetConnectFileContentOutput{}, err
	}

	th.Logger().Info(getConnectFileContentToolName+" tool called", "path", input.Path)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Getting Connect file content")
	th.Notify().Info("Retrieving Connect file content")
	result, err := client.GetConnectFileContent(ctx, input)
	if err != nil {
		LogRIError(th, "unable to retrieve Connect file content", err)
		return nil, GetConnectFileContentOutput{}, err
	}

	th.Logger().Info("Retrieved Connect file content successfully")
	th.Notify().Info("Retrieved Connect file content successfully")

	return nil, GetConnectFileContentOutput{Content: string(result)}, nil
}
