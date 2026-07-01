// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"fmt"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetConnectProjectsInput struct{}

func GetConnectProjects(ctx context.Context, req *mcp.CallToolRequest, input GetConnectProjectsInput) (*mcp.CallToolResult, rapididentity.GetConnectProjectsOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, rapididentity.GetConnectProjectsOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	result, err := client.GetConnectProjects(ctx)
	if err != nil {
		return nil, rapididentity.GetConnectProjectsOutput{}, err
	}

	return nil, *result, nil
}
