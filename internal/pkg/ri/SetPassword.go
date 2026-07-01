// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"fmt"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SetPasswordOutput struct {
	Result rapididentity.SetPasswordOutput `json:"result" jsonschema:"The set password result"`
}

func SetPassword(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.SetPasswordInput) (*mcp.CallToolResult, SetPasswordOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, SetPasswordOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	result, err := client.SetPassword(ctx, input)
	if err != nil {
		return nil, SetPasswordOutput{}, err
	}

	return nil, SetPasswordOutput{
		Result: result,
	}, nil
}
