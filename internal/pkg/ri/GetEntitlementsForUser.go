// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const searchEntitlementsForUserToolName = "search-entitlements-for-user"

type EntitlementForUserInput struct {
	Id string `json:"id" jsonschema:"The unique rapididentity id. Also known as the idautoID"`
}

type EntitlementForUserOutput struct {
	Resources            []Resource            `json:"resources" jsonschema:"The entitlement information such as name, id and requestId"`
	ResourceAssociations []ResourceAssociation `json:"resourceAssociations" jsonschema:"The entitlement resource associated with the user"`
}

type Resource struct {
	Id                   string `json:"id" jsonschema:"The unique entitlement resource id"`
	Name                 string `json:"name" jsonschema:"The friendly display name of the entitlement"`
	Description          string `json:"description" jsonschema:"The description of the entitlement"`
	Status               string `json:"status" jsonschema:"Whether the entitlement is requestable. The values will only be ACTIVE and INACTIVE"`
	DisableCertification bool   `json:"disableCertification" jsonschema:"Whether or not the entitlement can be certified or not"`
	NotUIRequestable     bool   `json:"notUIRequestable" jsonschema:"Whether or not the entitlement is requestable through the UI"`
	CanRequestExtend     bool   `json:"canRequestExtend" jsonschema:"Whether or not the entitlement can be extended"`
	CanRequestReset      bool   `json:"canRequestReset" jsonschema:"Whether or not the entitlement can be reset"`
}
type ResourceAssociation struct {
	UserId     string `json:"userId" jsonschema:"The unique rapididentity id. Also known as the idautoID"`
	RequestId  string `json:"requestId" jsonschema:"The latest id of the request. This is often used to populate previousRequestId in additional rapididentity api calls"`
	ResourceId string `json:"resourceId" jsonschema:"The unique entitlement resource id"`
	Status     string `json:"status" jsonschema:"The user association status. This can be one of GRANTED, REVOKED, NO_ASSOCIATION"`
}

func GetEntitlementForUser(ctx context.Context, req *mcp.CallToolRequest, input EntitlementForUserInput) (*mcp.CallToolResult, EntitlementForUserOutput, error) {
	client, th, err := ToolSetup(req, searchEntitlementsForUserToolName)
	if err != nil {
		return nil, EntitlementForUserOutput{}, err
	}

	th.Logger().Info(searchEntitlementsForUserToolName+" tool called", "userId", input.Id)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	path := fmt.Sprintf("workflow/users/%s/associations", input.Id)
	th.Logger().Info("Calling entitlement associations endpoint", "path", path)
	th.Notify().Info("Retrieving entitlements for user")
	entitlementAssociationsRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(th, "unable to retrieve entitlement associations", err)
		return nil, EntitlementForUserOutput{}, err
	}

	th.Logger().Debug("GET "+path+" response", "response", entitlementAssociationsRes)

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			th.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(entitlementAssociationsRes)

	entitlementAssociationsBody, err := io.ReadAll(entitlementAssociationsRes.Body)
	if err != nil {
		th.Logger().Error("unable to read response body for "+path+" response", "error", err, "status", entitlementAssociationsRes.StatusCode)
		return nil, EntitlementForUserOutput{}, err
	}

	th.Logger().Debug("GET "+path+" response body", "body", string(entitlementAssociationsBody))

	var output EntitlementForUserOutput

	err = json.Unmarshal(entitlementAssociationsBody, &output)
	if err != nil {
		th.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		return nil, EntitlementForUserOutput{}, err
	}

	th.Logger().Info("Retrieved entitlements successfully", "resourceCount", len(output.Resources), "associationCount", len(output.ResourceAssociations))
	th.Notify().Info(fmt.Sprintf("Retrieved %d entitlements for user", len(output.Resources)))

	return nil, output, nil
}
