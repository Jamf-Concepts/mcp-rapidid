package ri

import (
	"context"
	"fmt"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func GetConnectAction(ctx context.Context, req *mcp.CallToolRequest, input rapididentity.GetConnectActionByIdInput) (*mcp.CallToolResult, rapididentity.GetConnectActionByIdOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, rapididentity.GetConnectActionByIdOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	result, err := client.GetConnectActionById(ctx, input)
	if err != nil {
		return nil, rapididentity.GetConnectActionByIdOutput{}, err
	}

	return nil, *result, nil
}
