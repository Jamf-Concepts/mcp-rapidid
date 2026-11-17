// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, rapididentity.ConnectJob{}, fmt.Errorf("a job name is required")
	}

	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, getConnectJobToolName)
	if setupErr != nil {
		return nil, rapididentity.ConnectJob{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, getConnectJobToolName, start, &err)

	sh.Logger().Info(getConnectJobToolName+" tool called", "name", name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	existing, err := requireConnectJob(ctx, client, sh, name)
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

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(jobRes)

	if err = checkResponseStatus(jobRes); err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect job", err)
		return nil, rapididentity.ConnectJob{}, err
	}

	jobBody, err := io.ReadAll(jobRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for GET "+path+" response", "error", err, "status", jobRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", getConnectJobToolName), "see server logs", telemetry.ErrorCategoryThrownException)
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
