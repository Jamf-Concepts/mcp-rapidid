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

type SearchGroupsInput struct {
	Criteria string `json:"criteria" jsonschema:"The search criteria for the group."`
}

type SearchGroupsOutput struct {
	Users              []User  `json:"users" jsonschema:"A list of RapidIdentity users"`
	Groups             []Group `json:"groups" jsonschema:"A list of RapidIdentity groups"`
	AdminLimitEnforced bool    `json:"adminLimitEnforced" jsonschema:"Whether the admin limit was reached in the search"`
}

type Group struct {
	Id                  string   `json:"id" jsonschema:"The unique id for the group. Also known as the idautoID."`
	Version             int      `json:"version" jsonschema:"The version of the group."`
	Name                string   `json:"name" jsonschema:"The name of the group"`
	Description         string   `json:"The description of the group"`
	Dn                  string   `json:"dn" jsonschema:"The unique LDAP group identifier. Also known as the distinguishedName"`
	OwnerDNs            []string `json:"ownerDNs" jsonschema:"The unique LDAP user identifier for the users who are owners of the group"`
	CoOwnerDNs          []string `json:"coOwnerDNs" jsonschema:"The unique LDAP user identifier for the users who are managers of the group"`
	StaticMemberDNs     []string `json:"staticMemberDNs" jsonschema:"Users who have been made members of this group manually."`
	DynamicMemberFilter string   `json:"dynamicMemberFilter" jsonschema:"The LDAP filter that is evaluated to determine what users should be members of this group"`
	CreateDate          string   `json:"createDate" jsonschema:"The date this group was created in the format of yyyy-MM-ddThh:mm:ss.SSSZ"`
	ModifiedDate        string   `json:"modifiedDate" jsonschema:"The date this group was modified in the format of yyyy-MM-ddThh:mm:ss.SSSZ"`
}

func SearchGroups(ctx context.Context, req *mcp.CallToolRequest, input SearchGroupsInput) (*mcp.CallToolResult, SearchGroupsOutput, error) {
	options := GetRapidIdentityOptions()
	client, err := rapididentity.New(options)
	if err != nil {
		return nil, SearchGroupsOutput{}, err
	}
	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	path := fmt.Sprintf("roles/managedGroups/searchTask?criteria=%s", url.QueryEscape(input.Criteria))
	groupsRes, err := client.DoCustomRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, SearchGroupsOutput{}, err
	}
	defer func(r *http.Response) {
		err := r.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(groupsRes)

	resBody, err := io.ReadAll(groupsRes.Body)
	if err != nil {
		return nil, SearchGroupsOutput{}, err
	}

	var output SearchGroupsOutput
	err = json.Unmarshal(resBody, &output)
	if err != nil {
		return nil, SearchGroupsOutput{}, err
	}

	return nil, output, nil
}
