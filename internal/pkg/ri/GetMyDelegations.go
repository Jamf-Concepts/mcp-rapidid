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

const getMyDelegationsToolName = "get-my-delegations"

type GetMyDelegationsInput struct{}

type GetMyDelegationsOutput struct {
	Delegations []Delegation `json:"delegations" jsonschema:"A list of delegations accessible to the authenticated user"`
}

func GetMyDelegations(ctx context.Context, req *mcp.CallToolRequest, input GetMyDelegationsInput) (*mcp.CallToolResult, GetMyDelegationsOutput, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getMyDelegationsToolName)
	if setupErr != nil {
		return nil, GetMyDelegationsOutput{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getMyDelegationsToolName, start, &err)

	sh.Logger().Info(getMyDelegationsToolName + " tool called")

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	sh.Logger().Info("Calling profiles/delegations/my endpoint")
	delegationRes, err := client.DoCustomRequest(ctx, "GET", "profiles/delegations/my", nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve delegations", err)
		return nil, GetMyDelegationsOutput{}, err
	}

	sh.Logger().Debug("GET profiles/delegations/my response", "response", delegationRes)

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for profiles/delegations/my endpoint response", "error", err)
		}
	}(delegationRes)

	if err := checkResponseStatus(delegationRes); err != nil {
		LogRIError(ctx, sh, "unable to retrieve delegations", err)
		return nil, GetMyDelegationsOutput{}, err
	}

	delegationResBody, err := io.ReadAll(delegationRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for the profiles/delegations/my response", "error", err, "status", delegationRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", getMyDelegationsToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, GetMyDelegationsOutput{}, err
	}

	sh.Logger().Debug("GET profiles/delegations/my response body", "body", string(delegationResBody))

	var delegations []Delegation

	err = json.Unmarshal(delegationResBody, &delegations)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for GET profiles/delegations/my response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", getMyDelegationsToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, GetMyDelegationsOutput{}, err
	}

	sh.Logger().Info("Retrieved delegations successfully", "delegationCount", len(delegations))

	return nil, GetMyDelegationsOutput{Delegations: delegations}, nil
}
