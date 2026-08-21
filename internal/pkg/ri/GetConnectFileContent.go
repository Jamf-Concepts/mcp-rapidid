// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectFileContentToolName = "get-connect-file-content"

type GetConnectFileContentOutput struct {
	Content string `json:"content" jsonschema:"The text content of the Connect file"`
}

func GetConnectFileContent(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectFileContentInput) (*mcp.CallToolResult, GetConnectFileContentOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getConnectFileContentToolName)
	if setupErr != nil {
		return nil, GetConnectFileContentOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectFileContentToolName, start, &err)

	sh.Logger().Info(getConnectFileContentToolName+" tool called", "path", input.Path)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	sh.Logger().Info("Getting Connect file content")
	result, err := client.GetConnectFileContent(ctx, input)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect file content", err)
		return nil, GetConnectFileContentOutput{}, err
	}

	sh.Logger().Info("Retrieved Connect file content successfully")

	return nil, GetConnectFileContentOutput{Content: string(result)}, nil
}
