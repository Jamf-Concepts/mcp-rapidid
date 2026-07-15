// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const searchGroupsToolName = "search-groups"

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
	client, th, err := ToolSetup(req, searchGroupsToolName)
	if err != nil {
		return nil, SearchGroupsOutput{}, err
	}

	th.Logger().Info(searchGroupsToolName+" tool called", "criteria", input.Criteria)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	path := fmt.Sprintf("roles/managedGroups/searchTask?criteria=%s", url.QueryEscape(input.Criteria))

	th.Logger().Info("Searching groups", "path", path)
	th.Notify().Info("Searching for groups based on criteria")
	groupsRes, err := client.DoCustomRequest(ctx, "POST", path, nil)
	if err != nil {
		LogRIError(th, "unable to search groups", err)
		return nil, SearchGroupsOutput{}, err
	}

	th.Logger().Debug("POST "+path+" response", "response", groupsRes)

	defer func(r *http.Response) {
		if err := r.Body.Close(); err != nil {
			th.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(groupsRes)

	resBody, err := io.ReadAll(groupsRes.Body)
	if err != nil {
		th.Logger().Error("unable to read response body for "+path+" response", "error", err, "status", groupsRes.StatusCode)
		return nil, SearchGroupsOutput{}, err
	}

	th.Logger().Debug("POST "+path+" response body", "body", string(resBody))

	var output SearchGroupsOutput
	err = json.Unmarshal(resBody, &output)
	if err != nil {
		th.Logger().Error("unable to unmarshal json for POST "+path+" response body", "error", err)
		return nil, SearchGroupsOutput{}, err
	}

	th.Logger().Info("Retrieved groups successfully", "groupCount", len(output.Groups), "userCount", len(output.Users))
	th.Notify().Info(fmt.Sprintf("Retrieved %d groups", len(output.Groups)))

	return nil, output, nil
}
