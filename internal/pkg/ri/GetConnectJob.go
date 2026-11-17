// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const getConnectJobToolName = "get-connect-job"

type GetConnectJobInput struct {
	Name string `json:"name" jsonschema:"The name of the Connect job. Job names are unique across all projects"`
}

func GetConnectJob(ctx context.Context, req *mcp.CallToolRequest, input GetConnectJobInput) (*mcp.CallToolResult, rapididentity.ConnectJob, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getConnectJobToolName)
	if setupErr != nil {
		return nil, rapididentity.ConnectJob{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectJobToolName, start, &err)

	sh.Logger().Info(getConnectJobToolName+" tool called", "name", input.Name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	existing, err := requireConnectJob(ctx, client, sh, input.Name)
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}

	path := "admin/connect/jobs/" + url.PathEscape(existing.Id)

	sh.Logger().Info("Retrieving Connect job", "path", path)
	jobRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	jobBody, err := readConnectJobResponse(ctx, sh, getConnectJobToolName, jobRes)
	if err != nil {
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Debug("GET "+path+" response body", "body", string(jobBody))

	var job rapididentity.ConnectJob

	err = json.Unmarshal(jobBody, &job)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", getConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectJob{}, err
	}

	sh.Logger().Info("Retrieved Connect job successfully", "id", job.Id)

	return nil, job, nil
}
