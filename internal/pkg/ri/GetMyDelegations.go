// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetMyDelegationsInput struct{}

type GetMyDelegationsOutput struct {
	Delegations []Delegation `json:"delegations" jsonschema:"A list of delegations accessible to the authenticated user"`
}

func GetMyDelegations(ctx context.Context, req *mcp.CallToolRequest, input GetMyDelegationsInput) (*mcp.CallToolResult, GetMyDelegationsOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, GetMyDelegationsOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	delegationRes, err := client.DoCustomRequest(ctx, "GET", "profiles/delegations/my", nil)
	if err != nil {
		return nil, GetMyDelegationsOutput{}, err
	}

	defer func(res *http.Response) {
		err := res.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(delegationRes)

	delegationResBody, err := io.ReadAll(delegationRes.Body)
	if err != nil {
		return nil, GetMyDelegationsOutput{}, err
	}

	var delegations []Delegation

	err = json.Unmarshal(delegationResBody, &delegations)
	if err != nil {
		return nil, GetMyDelegationsOutput{}, err
	}

	return nil, GetMyDelegationsOutput{Delegations: delegations}, nil
}
