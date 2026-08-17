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

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const searchRapidIdentityUsersToolName = "search-users"

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

// reportingUsersResponse is the envelope returned by the GET /reporting/users
// endpoint, which wraps the user list rather than returning a bare array.
type reportingUsersResponse struct {
	Users              []User `json:"users"`
	AdminLimitEnforced bool   `json:"adminLimitEnforced"`
}

func SearchRapidIdentityUsers(ctx context.Context, req *mcp.CallToolRequest, input UserInput) (*mcp.CallToolResult, UserOutput, error) {
	var err error
	ctx, client, th, setupErr := ToolSetup(ctx, req, searchRapidIdentityUsersToolName)
	if setupErr != nil {
		return nil, UserOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, searchRapidIdentityUsersToolName, start, &err)

	th.Logger().Info(searchRapidIdentityUsersToolName+" tool called", "criteria", input.Criteria)

	defer func(c *rapididentity.Client) {
		if cerr := c.Close(); cerr != nil {
			LogRIError(ctx, th, "unable to close rapididentity client", cerr)
		}
	}(client)

	// Service Identities do not return results through the delegation-scoped
	// users search, so search via the reporting endpoint instead. Username and
	// password callers continue to use the delegation-based path below.
	if UsingServiceIdentity() {
		return searchUsersViaReporting(ctx, th, client, input)
	}

	th.Logger().Info("Calling profiles/delegations/my endpoint")
	th.Notify().Info("Retrieving delegations for caller")
	delegationRes, err := client.DoCustomRequest(ctx, "GET", "profiles/delegations/my", nil)
	if err != nil {
		LogRIError(ctx, th, "unable to retrieve delegations for user", err)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("GET profiles/delegations/my response", "response", delegationRes)

	defer func(res *http.Response) {
		if cerr := res.Body.Close(); cerr != nil {
			th.Logger().Warn("issue closing response body for profiles/delegations/my endpoint response", "error", cerr)
		}
	}(delegationRes)

	if err = CheckResponseStatus(delegationRes); err != nil {
		th.Logger().Error("unexpected status for GET profiles/delegations/my response", "status", delegationRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	delegationResBody, err := io.ReadAll(delegationRes.Body)
	if err != nil {
		th.Logger().Error("unable to read response body for the profiles/delegations/my response", "error", err, "status", delegationRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("GET profiles/delegations/my response body", "body", string(delegationResBody))

	var delegationOutputs []Delegation

	err = json.Unmarshal(delegationResBody, &delegationOutputs)
	if err != nil {
		th.Logger().Error("unable to unmarshal json for GET profiles/delegations/my response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("Unmarshaled delegation output", "delegations", delegationOutputs)
	th.Notify().Info(fmt.Sprintf("Retrieved %d delegations", len(delegationOutputs)))

	path := fmt.Sprintf("users?search=simple&criteria=%s", url.QueryEscape(input.Criteria))

	for _, delegationOutput := range delegationOutputs {
		path = fmt.Sprintf("%s&did=%s", path, url.QueryEscape(delegationOutput.Id))
	}

	th.Logger().Debug("Call GET users endpoint", "path", path)
	th.Logger().Info("Searching users across retrieved delegations")
	th.Notify().Info("Searching for users based on criteria")

	userRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, th, "unable to retrieve users based on supplied criteria", err)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("GET "+path+" response", "response", userRes)

	defer func(res *http.Response) {
		if cerr := res.Body.Close(); cerr != nil {
			th.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", cerr)
		}
	}(userRes)

	if err = CheckResponseStatus(userRes); err != nil {
		th.Logger().Error("unexpected status for GET "+path+" response", "status", userRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	userResBody, err := io.ReadAll(userRes.Body)
	if err != nil {
		th.Logger().Error("unable to read response body for the "+path+" response", "error", err, "status", userRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("GET "+path+" response body", "body", string(userResBody))

	var userOutputs []User

	err = json.Unmarshal(userResBody, &userOutputs)
	if err != nil {
		th.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("Unmarshaled user outputs", "users", userOutputs)
	th.Notify().Info(fmt.Sprintf("Retrieved %d users", len(userOutputs)))

	return nil, UserOutput{Users: userOutputs}, nil
}

// searchUsersViaReporting searches for users through the GET /reporting/users
// endpoint. This path is used when authenticating with a Service Identity,
// which does not return results through the delegation-scoped users search.
func searchUsersViaReporting(ctx context.Context, th *helper.ToolHelper, client *rapididentity.Client, input UserInput) (*mcp.CallToolResult, UserOutput, error) {
	path := fmt.Sprintf("reporting/users?criteria=%s", url.QueryEscape(input.Criteria))

	th.Logger().Debug("Call GET reporting/users endpoint", "path", path)
	th.Logger().Info("Searching users via the reporting endpoint for a service identity")
	th.Notify().Info("Searching for users based on criteria")

	userRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, th, "unable to retrieve users based on supplied criteria", err)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("GET "+path+" response", "response", userRes)

	defer func(res *http.Response) {
		if cerr := res.Body.Close(); cerr != nil {
			th.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", cerr)
		}
	}(userRes)

	if err = CheckResponseStatus(userRes); err != nil {
		th.Logger().Error("unexpected status for GET "+path+" response", "status", userRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	userResBody, err := io.ReadAll(userRes.Body)
	if err != nil {
		th.Logger().Error("unable to read response body for the "+path+" response", "error", err, "status", userRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("GET "+path+" response body", "body", string(userResBody))

	var reportingRes reportingUsersResponse

	err = json.Unmarshal(userResBody, &reportingRes)
	if err != nil {
		th.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf("RapidIdMcp.ToolError.%s", searchRapidIdentityUsersToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, UserOutput{}, err
	}

	th.Logger().Debug("Unmarshaled user outputs", "users", reportingRes.Users)
	th.Notify().Info(fmt.Sprintf("Retrieved %d users", len(reportingRes.Users)))

	return nil, UserOutput{Users: reportingRes.Users}, nil
}
