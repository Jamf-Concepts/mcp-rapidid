// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const startEntitlementRequestToolName = "start-entitlement-request"

type StartEntitlementRequestInput struct {
	RequestInfo []StartEntitlementRequestInfo `json:"requestInfo" jsonschema:"The request information to submit the entitlement request(s)"`
}

type StartEntitlementRequestInfo struct {
	Type              string `json:"type" jsonschema:"The request type. This can be GRANT or REVOKE"`
	UserId            string `json:"userId" jsonschema:"The unique rapididentity id for a user. Also known as the idautoID"`
	ResourceId        string `json:"resourceId" jsonschema:"The unique entitlement resourceId"`
	PreviousRequestId string `json:"previousRequestId" jsonschema:"The id of the previous request of the resource id. This can be found within the associations of a user"`
}

type StartEntitlementRequestOutput struct {
	RequestIds []string `json:"requestIds" jsonschema:"The requestIds from the initiated requests"`
}

type StartTaskPayload struct {
	RequestItems []StartTaskRequestItem `json:"requestItems" jsonschema:"The entitlements to request"`
}

type StartTaskRequestItem struct {
	Type              string `json:"type" jsonschema:"The request type. This can be a value of GRANT, GRANT_DENIED, REVOKE, REVOKE_DENIED"`
	RecipientId       string `json:"recipientId" jsonschema:"The unique rapididentity user id. Also known as the idautoID"`
	ResourceId        string `json:"resourceId" jsonschema:"The unique entitlement id to be requested"`
	PreviousRequestId string `json:"previousRequestId" jsonschema:"The request id of the previous request"`
}

func StartEntitlementRequest(ctx context.Context, req *mcp.CallToolRequest, input StartEntitlementRequestInput) (*mcp.CallToolResult, StartEntitlementRequestOutput, error) {
	client, th, err := ToolSetup(req, startEntitlementRequestToolName)
	if err != nil {
		return nil, StartEntitlementRequestOutput{}, err
	}

	th.Logger().Info(startEntitlementRequestToolName+" tool called", "requestCount", len(input.RequestInfo))

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	var payload StartTaskPayload

	for _, requestInfo := range input.RequestInfo {
		requestPayload := StartTaskRequestItem{
			Type:              requestInfo.Type,
			RecipientId:       requestInfo.UserId,
			ResourceId:        requestInfo.ResourceId,
			PreviousRequestId: requestInfo.PreviousRequestId,
		}

		payload.RequestItems = append(payload.RequestItems, requestPayload)
	}

	requestPayload, err := json.Marshal(payload)
	if err != nil {
		th.Logger().Error("unable to marshal entitlement request payload", "error", err)
		return nil, StartEntitlementRequestOutput{}, err
	}

	body := bytes.NewBuffer(requestPayload)

	th.Logger().Info("Calling workflow/tasks/startTask endpoint", "itemCount", len(payload.RequestItems))
	th.Notify().Info(fmt.Sprintf("Starting %d entitlement request(s)", len(payload.RequestItems)))
	startTaskRes, err := client.DoCustomRequest(ctx, "POST", "workflow/tasks/startTask", body)
	if err != nil {
		LogRIError(th, "unable to start entitlement task", err)
		return nil, StartEntitlementRequestOutput{}, err
	}

	th.Logger().Debug("POST workflow/tasks/startTask response", "response", startTaskRes)

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			th.Logger().Warn("issue closing response body for workflow/tasks/startTask endpoint response", "error", err)
		}
	}(startTaskRes)

	startTaskBody, err := io.ReadAll(startTaskRes.Body)
	if err != nil {
		th.Logger().Error("unable to read response body for workflow/tasks/startTask response", "error", err, "status", startTaskRes.StatusCode)
		return nil, StartEntitlementRequestOutput{}, err
	}

	th.Logger().Debug("POST workflow/tasks/startTask response body", "body", string(startTaskBody))

	var requestIds []string

	err = json.Unmarshal(startTaskBody, &requestIds)
	if err != nil {
		th.Logger().Error("unable to unmarshal json for POST workflow/tasks/startTask response body", "error", err)
		return nil, StartEntitlementRequestOutput{}, err
	}

	th.Logger().Info("Entitlement requests started successfully", "requestIdCount", len(requestIds))
	th.Notify().Info(fmt.Sprintf("Started %d entitlement request(s) successfully", len(requestIds)))

	return nil, StartEntitlementRequestOutput{
		RequestIds: requestIds,
	}, nil
}
