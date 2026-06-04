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

type GetGroupMembersInput struct {
	GroupId         string `json:"groupId" jsonschema:"The unique id of the group. Also known as the idautoID"`
	PageSize        int    `json:"pageSize" jsonschema:"The number of members to return in the query. This can be set to 1000 by default"`
	PagingSessionId string `json:"pagingSessionId" jsonschema:"The next page id. This value should be empty on the first query"`
}

type GetGroupMembersOutput struct {
	PagingSessionId      string   `json:"pagingSessionId" jsonschema:"The next page id. If empty, all pages have been read"`
	CalculatedMembership []string `json:"calculatedMembership" jsonschema:"A list of distinguishedName or dn of users who are members of the group. This will be in the format of idauto={idautoID},ou=Accounts,dc=meta"`
	TotalCount           int      `json:"totalCount" jsonschema:"The total number of members in the group"`
}

func GetGroupMembers(ctx context.Context, req *mcp.CallToolRequest, input GetGroupMembersInput) (*mcp.CallToolResult, GetGroupMembersOutput, error) {
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, GetGroupMembersOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	path := fmt.Sprintf("roles/groups/%s/membershipCalculation?pageSize=%d", input.GroupId, input.PageSize)
	if input.PagingSessionId != "" {
		path = fmt.Sprintf("%s&pagingSessionId=%s", path, input.PagingSessionId)
	}

	membersRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, GetGroupMembersOutput{}, err
	}

	defer func(res *http.Response) {
		err := res.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(membersRes)

	membersBody, err := io.ReadAll(membersRes.Body)
	if err != nil {
		return nil, GetGroupMembersOutput{}, err
	}

	var output GetGroupMembersOutput

	err = json.Unmarshal(membersBody, &output)
	if err != nil {
		return nil, GetGroupMembersOutput{}, err
	}

	return nil, output, nil
}
