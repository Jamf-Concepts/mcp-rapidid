// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, StartEntitlementRequestOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
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
		return nil, StartEntitlementRequestOutput{}, err
	}

	body := bytes.NewBuffer(requestPayload)

	startTaskRes, err := client.DoCustomRequest(ctx, "POST", "workflow/tasks/startTask", body)
	if err != nil {
		return nil, StartEntitlementRequestOutput{}, err
	}

	defer func(res *http.Response) {
		err := res.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(startTaskRes)

	startTaskBody, err := io.ReadAll(startTaskRes.Body)
	if err != nil {
		return nil, StartEntitlementRequestOutput{}, err
	}

	var requestIds []string

	err = json.Unmarshal(startTaskBody, &requestIds)
	if err != nil {
		return nil, StartEntitlementRequestOutput{}, err
	}

	return nil, StartEntitlementRequestOutput{
		RequestIds: requestIds,
	}, nil
}
