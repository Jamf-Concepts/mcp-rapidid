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

type UserInput struct {
	Criteria string `json:"criteria" jsonschema:"The search criteria for the user."`
}

type UserOutput struct {
	Users []User `json:"users" jsonschema:"A list of RapidIdentity users"`
}
type User struct {
	Id             string   `json:"id" jsonschema:"The unique rapididentity user identifier. Also known as the idautoID"`
	Dn             string   `json:"dn" jsonschema:"The unique LDAP user identifier. Also known as the distinguishName"`
	Username       string   `json:"username" jsonschema:"The username of the user"`
	FirstName      string   `json:"firstName" jsonschema:"The first name of the user"`
	LastName       string   `json:"lastName" jsonschema:"The last name of the user"`
	Email          string   `json:"email" jsonschema:"The email address of the user"`
	MobileNumbers  []string `json:"mobileNumbers" jsonschema:"The phone numbers of the user"`
	AlternateEmail string   `json:"alternateEmail " jsonschema:"The alternate email of the user"`
}

type Delegation struct {
	Id          string `json:"id" jsonschema:"The unique delegation id"`
	Name        string `json:"name" jsonschema:"The friendly display name of the delegation"`
	Description string `json:"description" jsonschema:"The description of the delegation"`
	Type        string `json:"type" jsonschema:"The delegation type. This is either MY or customer. a MY delegation is for viewing your own user data while CUSTOM is for viewing other users' data'"`
}

func SearchRapidIdentityUsers(ctx context.Context, req *mcp.CallToolRequest, input UserInput) (*mcp.CallToolResult, UserOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, UserOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	delegationRes, err := client.DoCustomRequest(ctx, "GET", "profiles/delegations/my", nil)
	if err != nil {
		return nil, UserOutput{}, err
	}

	defer func(res *http.Response) {
		err := res.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(delegationRes)

	delegationResBody, err := io.ReadAll(delegationRes.Body)
	if err != nil {
		return nil, UserOutput{}, err
	}

	var delegationOutputs []Delegation

	err = json.Unmarshal(delegationResBody, &delegationOutputs)
	if err != nil {
		return nil, UserOutput{}, err
	}

	path := fmt.Sprintf("users?search=simple&criteria=%s", url.QueryEscape(input.Criteria))

	for _, delegationOutput := range delegationOutputs {
		path = fmt.Sprintf("%s&did=%s", path, url.QueryEscape(delegationOutput.Id))
	}

	userRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, UserOutput{}, err
	}

	defer func(res *http.Response) {
		err := res.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(userRes)

	userResBody, err := io.ReadAll(userRes.Body)
	if err != nil {
		return nil, UserOutput{}, err
	}

	var userOutputs []User

	err = json.Unmarshal(userResBody, &userOutputs)
	if err != nil {
		return nil, UserOutput{}, err
	}

	return nil, UserOutput{Users: userOutputs}, nil
}
