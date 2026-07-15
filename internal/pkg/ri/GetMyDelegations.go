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

const getMyDelegationsToolName = "get-my-delegations"

type GetMyDelegationsInput struct{}

type GetMyDelegationsOutput struct {
	Delegations []Delegation `json:"delegations" jsonschema:"A list of delegations accessible to the authenticated user"`
}

func GetMyDelegations(ctx context.Context, req *mcp.CallToolRequest, input GetMyDelegationsInput) (*mcp.CallToolResult, GetMyDelegationsOutput, error) {
	client, th, err := ToolSetup(req, getMyDelegationsToolName)
	if err != nil {
		return nil, GetMyDelegationsOutput{}, err
	}

	th.Logger().Info(getMyDelegationsToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(th, "unable to close rapididentity client", err)
		}
	}(client)

	th.Logger().Info("Calling profiles/delegations/my endpoint")
	th.Notify().Info("Retrieving delegations")
	delegationRes, err := client.DoCustomRequest(ctx, "GET", "profiles/delegations/my", nil)
	if err != nil {
		LogRIError(th, "unable to retrieve delegations", err)
		return nil, GetMyDelegationsOutput{}, err
	}

	th.Logger().Debug("GET profiles/delegations/my response", "response", delegationRes)

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			th.Logger().Warn("issue closing response body for profiles/delegations/my endpoint response", "error", err)
		}
	}(delegationRes)

	delegationResBody, err := io.ReadAll(delegationRes.Body)
	if err != nil {
		th.Logger().Error("unable to read response body for the profiles/delegations/my response", "error", err, "status", delegationRes.StatusCode)
		return nil, GetMyDelegationsOutput{}, err
	}

	th.Logger().Debug("GET profiles/delegations/my response body", "body", string(delegationResBody))

	var delegations []Delegation

	err = json.Unmarshal(delegationResBody, &delegations)
	if err != nil {
		th.Logger().Error("unable to unmarshal json for GET profiles/delegations/my response body", "error", err)
		return nil, GetMyDelegationsOutput{}, err
	}

	th.Logger().Info("Retrieved delegations successfully", "delegationCount", len(delegations))
	th.Notify().Info(fmt.Sprintf("Retrieved %d delegations", len(delegations)))

	return nil, GetMyDelegationsOutput{Delegations: delegations}, nil
}
