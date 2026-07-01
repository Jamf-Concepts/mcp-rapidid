// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"fmt"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func SaveConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.SaveConnectActionInput) (*mcp.CallToolResult, rapididentity.SaveConnectActionOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, rapididentity.SaveConnectActionOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	result, err := client.SaveConnectAction(ctx, input)
	if err != nil {
		return nil, rapididentity.SaveConnectActionOutput{}, err
	}

	return nil, *result, nil
}
