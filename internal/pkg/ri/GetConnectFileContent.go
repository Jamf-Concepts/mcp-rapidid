// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"fmt"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetConnectFileContentOutput struct {
	Content string `json:"content" jsonschema:"The text content of the Connect file"`
}

func GetConnectFileContent(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectFileContentInput) (*mcp.CallToolResult, GetConnectFileContentOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, GetConnectFileContentOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	result, err := client.GetConnectFileContent(ctx, input)
	if err != nil {
		return nil, GetConnectFileContentOutput{}, err
	}

	return nil, GetConnectFileContentOutput{Content: string(result)}, nil
}
