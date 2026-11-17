// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

// findConnectJob returns the job named name, or nil when there is none. Job
// names are unique across all projects, but only when compared exactly: the
// server refuses a second "xyz1JobTest" ("Duplicate job name") yet accepts
// "XYZ1JOBTEST" next to it (observed against a live tenant). So an exact
// match wins; otherwise a single match ignoring case is used, and several
// matches ignoring case are an error rather than a guess.
func findConnectJob(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, name string) (*rapididentity.ConnectJob, error) {
	sh.Logger().Info("Looking up the Connect job by name")
	jobs, err := client.GetConnectJobs(ctx, rapididentity.GetConnectJobsInput{})
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve rapididentity connect jobs", err)
		return nil, err
	}
	var match *rapididentity.ConnectJob
	var matchNames []string
	for i := range jobs.Jobs {
		if jobs.Jobs[i].Name == name {
			return &jobs.Jobs[i], nil
		}
		if strings.EqualFold(jobs.Jobs[i].Name, name) {
			match = &jobs.Jobs[i]
			matchNames = append(matchNames, jobs.Jobs[i].Name)
		}
	}
	if len(matchNames) > 1 {
		err = fmt.Errorf("more than one Connect job matches %q ignoring case (%s); give the exact name", name, strings.Join(matchNames, ", "))
		LogRIError(ctx, sh, "Connect job request refused", err)
		return nil, err
	}
	return match, nil
}

// requireConnectJob trims name, refuses a blank one and returns the existing
// job from the job list, or an error naming the missing job.
//
// Callers address the job by the returned job's id. The job endpoints take
// "nameOrId", but a bare job name is not found (404 "Couldn't find job");
// only the id or "<project>.<name>" work (observed against a live tenant).
func requireConnectJob(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, name string) (*rapididentity.ConnectJob, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		err := fmt.Errorf("a job name is required")
		LogRIError(ctx, sh, "Connect job request refused", err)
		return nil, err
	}
	job, err := findConnectJob(ctx, client, sh, name)
	if err != nil {
		return nil, err
	}
	if job == nil {
		err = fmt.Errorf("no Connect job named %q exists", name)
		LogRIError(ctx, sh, "Connect job request refused", err)
		return nil, err
	}
	return job, nil
}

// connectErrorBody is the body Connect sends with an error status, for
// example {"success":false,"message":"Duplicate job name","httpStatus":500}.
type connectErrorBody struct {
	Message string `json:"message"`
}

// readConnectJobResponse closes the response, rejects a non-2xx status and
// returns the body. A rejection carries the server's message when there is
// one, such as "The data you provided is out of date" for a 409 (another
// change to the job got in first).
func readConnectJobResponse(ctx context.Context, sh *helper.ServerHelper, toolName string, res *http.Response) ([]byte, error) {
	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for Connect job endpoint response", "error", err)
		}
	}(res)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for Connect job endpoint response", "error", err, "status", res.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", toolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, err
	}

	if err := checkResponseStatus(res); err != nil {
		var errorBody connectErrorBody
		if json.Unmarshal(body, &errorBody) == nil && errorBody.Message != "" {
			err = fmt.Errorf("%w: %s", err, errorBody.Message)
		}
		LogRIError(ctx, sh, "Connect job request failed", err)
		return nil, err
	}

	return body, nil
}
