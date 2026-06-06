package ri

import (
	"context"
	"fmt"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func GetConnectActions(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectActionsInput) (*mcp.CallToolResult, rapididentity.GetConnectActionsOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, rapididentity.GetConnectActionsOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	result, err := client.GetConnectActions(ctx, input)
	if err != nil {
		return nil, rapididentity.GetConnectActionsOutput{}, err
	}

	return nil, *result, nil
}
