// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"fmt"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const connect_log_name = "connect-tools"

type GetConnectProjectsInput struct{}

func GetConnectProjects(ctx context.Context, req *mcp.CallToolRequest, input GetConnectProjectsInput) (*mcp.CallToolResult, rapididentity.GetConnectProjectsOutput, error) {
	client, th, err := ToolSetup(req, connect_log_name)
	if err != nil {
		return nil, rapididentity.GetConnectProjectsOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			LogRIError(th, "unable to close connection to rapididentity", err)
		}
	}(client)

	
	th.Notify().Info("Calling RapidIdentity Connect projects endpoint")
	result, err := client.GetConnectProjects(ctx)
	if err != nil {
		LogRIError(th, "unable to retrieve rapididentity connect projects", err)
		return nil, rapididentity.GetConnectProjectsOutput{}, err
	}

	th.Notify().Info(fmt.Sprintf("Retrieved %d projects", len(result.Projects)))

	return nil, *result, nil
}
