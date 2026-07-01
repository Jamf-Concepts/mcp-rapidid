// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"fmt"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RunConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.RunConnectActionInput) (*mcp.CallToolResult, rapididentity.RunConnectActionOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, rapididentity.RunConnectActionOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	result, err := client.RunConnectAction(ctx, input)
	if err != nil {
		return nil, rapididentity.RunConnectActionOutput{}, err
	}

	return nil, *result, nil
}
