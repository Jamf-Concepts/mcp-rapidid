// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// saveConnectProject backs create-connect-project and update-connect-project.
//
// Both tools go through POST /admin/connect/projects, which creates or
// updates and behaves like this (observed against a live tenant):
//
//   - The project is keyed by name, not id: the server runs
//     "update connect_projects ... where name=?". A body without a name does
//     not update anything (it fails on the unique id constraint).
//   - A POST whose name matches an existing project REPLACES that project.
//     Every field missing from the body is cleared (description, groups,
//     restPoints), and the project is given a new id unless the body carries
//     the existing one. There is no "already exists" error.
//
// So the server never protects a project from being overwritten, and these
// tools must:
//
//   - create (mustExist false) fails when a project with the name exists,
//     instead of silently wiping it;
//   - update (mustExist true) fails when no project has the name, instead of
//     silently creating one, and sends back the WHOLE current project, with
//     its id, so that only the fields in changes are altered. The current
//     project is kept as raw JSON per field (map[string]json.RawMessage), not
//     decoded into rapididentity.ConnectProject, so that fields the SDK type
//     does not model are sent back unchanged instead of being dropped.
//
// Names are compared ignoring case; the existing project's own spelling is
// what gets sent, because the server matches names exactly.
func saveConnectProject(ctx context.Context, req *mcp.CallToolRequest, toolName string, name string, mustExist bool, changes connectProjectChanges) (*mcp.CallToolResult, rapididentity.ConnectProject, error) {
	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, toolName)
	if setupErr != nil {
		return nil, rapididentity.ConnectProject{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, toolName, start, &err)

	sh.Logger().Info(toolName+" tool called", "name", name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	name = strings.TrimSpace(name)
	if name == "" {
		err = fmt.Errorf("a project name is required")
		LogRIError(ctx, sh, "unable to save Connect project", err)
		return nil, rapididentity.ConnectProject{}, err
	}
	if mustExist && changes == (connectProjectChanges{}) {
		err = fmt.Errorf("nothing to update: pass at least one field to change")
		LogRIError(ctx, sh, "unable to save Connect project", err)
		return nil, rapididentity.ConnectProject{}, err
	}

	existingName, err := findConnectProjectName(ctx, client, sh, name)
	if err != nil {
		return nil, rapididentity.ConnectProject{}, err
	}

	switch {
	case !mustExist && existingName != "":
		err = fmt.Errorf("a Connect project named %q already exists", existingName)
	case mustExist && existingName == "":
		err = fmt.Errorf("no Connect project named %q exists", name)
	}
	if err != nil {
		LogRIError(ctx, sh, "unable to save Connect project", err)
		return nil, rapididentity.ConnectProject{}, err
	}

	var payload map[string]json.RawMessage
	if existingName != "" {
		payload, err = getConnectProjectJSON(ctx, client, sh, toolName, existingName)
	} else {
		var nameJSON json.RawMessage
		nameJSON, err = json.Marshal(name)
		payload = map[string]json.RawMessage{"name": nameJSON}
	}
	if err != nil {
		return nil, rapididentity.ConnectProject{}, err
	}
	changed, err := changes.rawFields()
	if err != nil {
		return nil, rapididentity.ConnectProject{}, err
	}
	for field, value := range changed {
		payload[field] = value
	}

	requestPayload, err := json.Marshal(payload)
	if err != nil {
		sh.Logger().Error("unable to marshal Connect project payload", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", toolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectProject{}, err
	}

	body := bytes.NewBuffer(requestPayload)

	sh.Logger().Info("Calling admin/connect/projects endpoint")
	projectRes, err := client.DoCustomRequest(ctx, "POST", "admin/connect/projects", body)
	if err != nil {
		LogRIError(ctx, sh, "unable to save Connect project", err)
		return nil, rapididentity.ConnectProject{}, err
	}

	sh.Logger().Debug("POST admin/connect/projects response", "response", projectRes)

	projectBody, err := readConnectProjectResponse(ctx, sh, toolName, projectRes)
	if err != nil {
		return nil, rapididentity.ConnectProject{}, err
	}

	sh.Logger().Debug("POST admin/connect/projects response body", "body", string(projectBody))

	var project rapididentity.ConnectProject

	err = json.Unmarshal(projectBody, &project)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for POST admin/connect/projects response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", toolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.ConnectProject{}, err
	}

	sh.Logger().Info("Saved Connect project successfully", "id", project.Id)

	return nil, project, nil
}

// connectProjectChanges holds the fields that create and update set. A nil
// field is left as it is; a pointer to "" clears the field. The JSON names
// are the server's field names.
type connectProjectChanges struct {
	Description     *string `json:"description,omitempty"`
	AdminGroupDN    *string `json:"adminGroupDN,omitempty"`
	OperatorGroupDN *string `json:"operatorGroupDN,omitempty"`
	AuditorGroupDN  *string `json:"auditorGroupDN,omitempty"`
}

// rawFields returns the fields that are set, keyed by their JSON names and
// encoded as raw JSON, ready to be merged into a project body.
func (c connectProjectChanges) rawFields() (map[string]json.RawMessage, error) {
	encoded, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// findConnectProjectName returns the exact name of the project whose name
// matches name ignoring case, or "" when there is none. The server matches
// names exactly, so callers send the returned spelling, not the caller's.
func findConnectProjectName(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, name string) (string, error) {
	sh.Logger().Info("Checking for an existing Connect project with the same name")
	projects, err := client.GetConnectProjects(ctx)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve rapididentity connect projects", err)
		return "", err
	}
	for _, project := range projects.Projects {
		if strings.EqualFold(project.Name, name) {
			return project.Name, nil
		}
	}
	return "", nil
}

// getConnectProjectJSON reads one project, by its exact name, as raw JSON.
func getConnectProjectJSON(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, toolName string, name string) (map[string]json.RawMessage, error) {
	path := "admin/connect/projects/" + url.PathEscape(name)

	sh.Logger().Info("Retrieving the current Connect project", "path", path)
	res, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect project", err)
		return nil, err
	}

	projectBody, err := readConnectProjectResponse(ctx, sh, toolName, res)
	if err != nil {
		return nil, err
	}

	var project map[string]json.RawMessage
	if err = json.Unmarshal(projectBody, &project); err != nil || project == nil {
		if err == nil {
			err = fmt.Errorf("empty response retrieving Connect project %q", name)
		}
		sh.Logger().Error("unable to unmarshal json for GET "+path+" response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", toolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, err
	}

	return project, nil
}

// readConnectProjectResponse closes the response, rejects a non-2xx status and
// returns the body.
func readConnectProjectResponse(ctx context.Context, sh *helper.ServerHelper, toolName string, res *http.Response) ([]byte, error) {
	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for admin/connect/projects endpoint response", "error", err)
		}
	}(res)

	if err := checkResponseStatus(res); err != nil {
		LogRIError(ctx, sh, "Connect project request failed", err)
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for admin/connect/projects response", "error", err, "status", res.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", toolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, err
	}

	return body, nil
}
