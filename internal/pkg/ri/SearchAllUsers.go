// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SearchAllUsersInput struct {
	Criteria string `json:"criteria" jsonschema:"The search criteria for the user (name, username, email, etc.)"`
}

type SearchAllUsersOutput struct {
	Users              []User `json:"users" jsonschema:"A list of RapidIdentity users matching the criteria"`
	AdminLimitEnforced bool   `json:"adminLimitEnforced" jsonschema:"Whether the server-side administrative result limit was enforced"`
}

func SearchAllUsers(ctx context.Context, req *mcp.CallToolRequest, input SearchAllUsersInput) (*mcp.CallToolResult, SearchAllUsersOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, SearchAllUsersOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	path := fmt.Sprintf("reporting/users?criteria=%s", url.QueryEscape(input.Criteria))

	userRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, SearchAllUsersOutput{}, err
	}

	defer func(res *http.Response) {
		err := res.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(userRes)

	userResBody, err := io.ReadAll(userRes.Body)
	if err != nil {
		return nil, SearchAllUsersOutput{}, err
	}

	var output SearchAllUsersOutput

	err = json.Unmarshal(userResBody, &output)
	if err != nil {
		return nil, SearchAllUsersOutput{}, err
	}

	return nil, output, nil
}
