// Copyright 2026, Jamf Software LLC

package ri

import (
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

const searchEntitlementsForUserToolName = "search-entitlements-for-user"

type EntitlementForUserInput struct {
	Id string `json:"id" jsonschema:"The unique rapididentity id. Also known as the idautoID"`
}

type EntitlementForUserOutput struct {
	Resources            []Resource            `json:"resources" jsonschema:"The entitlement information such as name, id and requestId"`
	ResourceAssociations []ResourceAssociation `json:"resourceAssociations" jsonschema:"The entitlement resource associated with the user"`
}

type Resource struct {
	Id                   string   `json:"id" jsonschema:"The unique entitlement resource id"`
	Name                 string   `json:"name" jsonschema:"The friendly display name of the entitlement"`
	Description          string   `json:"description" jsonschema:"The description of the entitlement"`
	Status               string   `json:"status" jsonschema:"Whether the entitlement is requestable. The values will only be ACTIVE and INACTIVE"`
	OwnerIds             []string `json:"ownerIds" jsonschema:"The list of user ids that own the entitlement"`
	DisableCertification bool     `json:"disableCertification" jsonschema:"Whether or not the entitlement can be certified or not"`
	NotUIRequestable     bool     `json:"notUIRequestable" jsonschema:"Whether or not the entitlement is requestable through the UI"`
	CanRequestExtend     bool     `json:"canRequestExtend" jsonschema:"Whether or not the entitlement can be extended"`
	CanRequestReset      bool     `json:"canRequestReset" jsonschema:"Whether or not the entitlement can be reset"`
}

type ApprovalHistoryEntry struct {
	Type          string `json:"type" jsonschema:"The type of approval action, e.g. APPROVED"`
	ResponderId   string `json:"responderId" jsonschema:"The unique id of the user who responded to the approval"`
	ResponderName string `json:"responderName" jsonschema:"The display name of the user who responded to the approval"`
	ResponseDate  string `json:"responseDate" jsonschema:"The date the approver responded"`
	AssignedDate  string `json:"assignedDate" jsonschema:"The date the approval task was assigned"`
}

type ResourceAssociation struct {
	UserId             string                 `json:"userId" jsonschema:"The unique rapididentity id. Also known as the idautoID"`
	RequestId          string                 `json:"requestId" jsonschema:"The latest id of the request. This is often used to populate previousRequestId in additional rapididentity api calls"`
	ResourceId         string                 `json:"resourceId" jsonschema:"The unique entitlement resource id"`
	Status             string                 `json:"status" jsonschema:"The user association status. This can be one of GRANTED, REVOKED, NO_ASSOCIATION"`
	GrantDate          string                 `json:"workflowEndDate" jsonschema:"The date the entitlement was granted to the user"`
	ExpirationDateTime string                 `json:"expirationDateTime" jsonschema:"The date and time when the entitlement expires"`
	ApprovalHistory    []ApprovalHistoryEntry `json:"approvalHistory" jsonschema:"The list of approval actions taken on the entitlement grant"`
}

func GetEntitlementForUser(ctx context.Context, req *mcp.CallToolRequest, input EntitlementForUserInput) (*mcp.CallToolResult, EntitlementForUserOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, searchEntitlementsForUserToolName)
	if setupErr != nil {
		return nil, EntitlementForUserOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, searchEntitlementsForUserToolName, start, &err)

	sh.Logger().Info(searchEntitlementsForUserToolName+" tool called", "userId", input.Id)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	path := fmt.Sprintf("workflow/users/%s/associations", input.Id)
	sh.Logger().Info("Calling entitlement associations endpoint", "path", path)
	entitlementAssociationsRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve entitlement associations", err)
		return nil, EntitlementForUserOutput{}, err
	}

	sh.Logger().Debug("GET "+path+" response", "response", entitlementAssociationsRes)

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(entitlementAssociationsRes)

	if err := checkResponseStatus(entitlementAssociationsRes); err != nil {
		LogRIError(ctx, sh, "unable to retrieve entitlement associations", err)
		return nil, EntitlementForUserOutput{}, err
	}

	entitlementAssociationsBody, err := io.ReadAll(entitlementAssociationsRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for "+path+" response", "error", err, "status", entitlementAssociationsRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", searchEntitlementsForUserToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, EntitlementForUserOutput{}, err
	}

	sh.Logger().Debug("GET "+path+" response body", "body", string(entitlementAssociationsBody))

	var output EntitlementForUserOutput

	err = json.Unmarshal(entitlementAssociationsBody, &output)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", searchEntitlementsForUserToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, EntitlementForUserOutput{}, err
	}

	sh.Logger().Info("Retrieved entitlements successfully", "resourceCount", len(output.Resources), "associationCount", len(output.ResourceAssociations))

	return nil, output, nil
}
