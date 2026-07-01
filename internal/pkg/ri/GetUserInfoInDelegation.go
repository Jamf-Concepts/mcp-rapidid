// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UserInfoInDelegationInput struct {
	DelegationId string `json:"delegationId" jsonschema:"The unique delegation id to search the criteria in"`
	Filter       string `json:"filter" jsonschema:"The LDAP filter to use for searching"`
}

type UserInfoInDelegationOutput struct {
	AdminLimitEnforced bool             `json:"adminLimitEnforced" jsonschema:"Whether the admin limit was reached in the search"`
	Profiles           []DelegationUser `json:"profiles" jsonschema:"The users returned from the search criteria"`
}
type DelegationUser struct {
	Id         string                    `json:"id" jsonschema:"The unique rapididentity user identifier. Also known as the idautoID"`
	Dn         string                    `json:"dn" jsonschema:"The unique LDAP user identifier. Also known as the distinguishName"`
	Attributes []DelegationUserAttribute `json:"attributes" jsonschema:"The attributes and their values associated with the user"`
}

type DelegationUserAttribute struct {
	Id     string   `json:"id" jsonschema:"The unique id for the attribute"`
	Name   string   `json:"name" jsonschema:"The friendly display name of the attribute"`
	Values []string `json:"values" jsonschema:"The value or values defined for that attribute"`
}

func GetUserInfoInDelegation(ctx context.Context, req *mcp.CallToolRequest, input UserInfoInDelegationInput) (*mcp.CallToolResult, UserInfoInDelegationOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, UserInfoInDelegationOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	path := fmt.Sprintf("profiles/delegations/my/%s/profiles/searchByFilter", input.DelegationId)
	body := bytes.NewBufferString(input.Filter)
	headers := http.Header{}
	headers.Set("Content-Type", "text/plain")

	profilesRes, err := client.DoCustomRequestWithHeaders(ctx, "POST", path, headers, body)
	if err != nil {
		return nil, UserInfoInDelegationOutput{}, err
	}

	defer func(res *http.Response) {
		err := res.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(profilesRes)

	profilesBody, err := io.ReadAll(profilesRes.Body)
	if err != nil {
		return nil, UserInfoInDelegationOutput{}, err
	}

	var output UserInfoInDelegationOutput

	err = json.Unmarshal(profilesBody, &output)
	if err != nil {
		return nil, UserInfoInDelegationOutput{}, err
	}

	return nil, output, nil
}
