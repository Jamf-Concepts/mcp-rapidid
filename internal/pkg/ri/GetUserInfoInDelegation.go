// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getUserInfoInDelegationToolName = "get-user-info-in-delegation"

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
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getUserInfoInDelegationToolName)
	if setupErr != nil {
		return nil, UserInfoInDelegationOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getUserInfoInDelegationToolName, start, &err)

	sh.Logger().Info(getUserInfoInDelegationToolName+" tool called", "delegationId", input.DelegationId)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	path := fmt.Sprintf("profiles/delegations/my/%s/profiles/searchByFilter", input.DelegationId)
	body := bytes.NewBufferString(input.Filter)
	headers := http.Header{}
	headers.Set("Content-Type", "text/plain")

	sh.Logger().Info("Searching users by filter in delegation", "path", path)
	profilesRes, err := client.DoCustomRequestWithHeaders(ctx, "POST", path, headers, body)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve user info in delegation", err)
		return nil, UserInfoInDelegationOutput{}, err
	}

	sh.Logger().Debug("POST "+path+" response", "response", profilesRes)

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(profilesRes)

	if err := checkResponseStatus(profilesRes); err != nil {
		LogRIError(ctx, sh, "unable to retrieve user info in delegation", err)
		return nil, UserInfoInDelegationOutput{}, err
	}

	profilesBody, err := io.ReadAll(profilesRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for "+path+" response", "error", err, "status", profilesRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", getUserInfoInDelegationToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserInfoInDelegationOutput{}, err
	}

	sh.Logger().Debug("POST "+path+" response body", "body", string(profilesBody))

	var output UserInfoInDelegationOutput

	err = json.Unmarshal(profilesBody, &output)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for POST "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", getUserInfoInDelegationToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserInfoInDelegationOutput{}, err
	}

	sh.Logger().Info("Retrieved user info in delegation successfully", "profileCount", len(output.Profiles))

	return nil, output, nil
}
