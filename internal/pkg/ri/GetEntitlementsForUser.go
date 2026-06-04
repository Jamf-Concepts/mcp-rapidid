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
	options := GetRapidIdentityOptions()

	client, err := rapididentity.New(options)
	if err != nil {
		return nil, EntitlementForUserOutput{}, err
	}

	defer func(c *rapididentity.Client) {
		err = c.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(client)

	path := fmt.Sprintf("workflow/users/%s/associations", input.Id)
	entitlementAssociationsRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, EntitlementForUserOutput{}, err
	}

	defer func(res *http.Response) {
		err := res.Body.Close()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}(entitlementAssociationsRes)

	entitlementAssociationsBody, err := io.ReadAll(entitlementAssociationsRes.Body)

	var output EntitlementForUserOutput

	err = json.Unmarshal(entitlementAssociationsBody, &output)
	if err != nil {
		return nil, EntitlementForUserOutput{}, err
	}

	return nil, output, nil
}
