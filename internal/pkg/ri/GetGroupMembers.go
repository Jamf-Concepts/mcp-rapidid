// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getGroupMembersToolName = "get-group-members"

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
	var err error
	ctx, client, th, setupErr := ToolSetup(ctx, req, getGroupMembersToolName)
	if setupErr != nil {
		return nil, GetGroupMembersOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getGroupMembersToolName, start, &err)

	th.Logger().Info(getGroupMembersToolName+" tool called", "groupId", input.GroupId, "pageSize", input.PageSize)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, th, "unable to close rapididentity client", err)
		}
	}(client)

	path := fmt.Sprintf("roles/groups/%s/membershipCalculation?pageSize=%d", url.PathEscape(input.GroupId), input.PageSize)
	if input.PagingSessionId != "" {
		path = fmt.Sprintf("%s&pagingSessionId=%s", path, url.QueryEscape(input.PagingSessionId))
	}

	th.Logger().Info("Getting group members", "path", path)
	th.Notify().Info("Retrieving group members")
	membersRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, th, "unable to retrieve group members", err)
		return nil, GetGroupMembersOutput{}, err
	}

	th.Logger().Debug("GET "+path+" response", "response", membersRes)

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			th.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(membersRes)

	if err = CheckResponseStatus(membersRes); err != nil {
		th.Logger().Error("unexpected status for GET "+path+" response", "status", membersRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", getGroupMembersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, GetGroupMembersOutput{}, err
	}

	membersBody, err := io.ReadAll(membersRes.Body)
	if err != nil {
		th.Logger().Error("unable to read response body for "+path+" response", "error", err, "status", membersRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", getGroupMembersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, GetGroupMembersOutput{}, err
	}

	th.Logger().Debug("GET "+path+" response body", "body", string(membersBody))

	var output GetGroupMembersOutput

	err = json.Unmarshal(membersBody, &output)
	if err != nil {
		th.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", getGroupMembersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, GetGroupMembersOutput{}, err
	}

	th.Logger().Info("Retrieved group members successfully", "totalCount", output.TotalCount)
	th.Notify().Info(fmt.Sprintf("Retrieved %d group members", output.TotalCount))

	return nil, output, nil
}
