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

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const updateConnectProjectToolName = "update-connect-project"

// UpdateConnectProjectInput uses pointers so that a field left out ("keep it")
// can be told apart from an empty string ("clear it").
type UpdateConnectProjectInput struct {
	Name            string  `json:"name" jsonschema:"The name of the existing Connect project to update. Projects cannot be renamed"`
	Description     *string `json:"description,omitempty" jsonschema:"The new description of the project. Leave out to keep the current one, or pass an empty string to clear it"`
	AdminGroupDN    *string `json:"adminGroupDN,omitempty" jsonschema:"The LDAP distinguished name (DN) of the project's admin group. Leave out to keep the current one, or pass an empty string to clear it"`
	OperatorGroupDN *string `json:"operatorGroupDN,omitempty" jsonschema:"The LDAP distinguished name (DN) of the project's operator group. Leave out to keep the current one, or pass an empty string to clear it"`
	AuditorGroupDN  *string `json:"auditorGroupDN,omitempty" jsonschema:"The LDAP distinguished name (DN) of the project's auditor group. Leave out to keep the current one, or pass an empty string to clear it"`
}

// UpdateConnectProject changes the given fields of an existing project and
// keeps every other field as it is. It refuses a name that no project has,
// because the server would otherwise create a new project. The server
// replaces the whole project on update, so the current project is read into
// ConnectProject, changed and posted back with its id. The read is strict: a
// field ConnectProject does not have stops the update, so that it is never
// silently cleared. See ConnectProjects.go.
func UpdateConnectProject(ctx context.Context, req *mcp.CallToolRequest, input UpdateConnectProjectInput) (*mcp.CallToolResult, ConnectProject, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ConnectProject{}, fmt.Errorf("a project name is required")
	}
	if input.Description == nil && input.AdminGroupDN == nil && input.OperatorGroupDN == nil && input.AuditorGroupDN == nil {
		return nil, ConnectProject{}, fmt.Errorf("nothing to update: pass at least one field to change")
	}

	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, updateConnectProjectToolName)
	if setupErr != nil {
		return nil, ConnectProject{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, updateConnectProjectToolName, start, &err)

	sh.Logger().Info(updateConnectProjectToolName+" tool called", "name", name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	existingName, err := findConnectProjectName(ctx, client, sh, name)
	if err != nil {
		return nil, ConnectProject{}, err
	}
	if existingName == "" {
		err = fmt.Errorf("no Connect project named %q exists", name)
		LogRIError(ctx, sh, "unable to update Connect project", err)
		return nil, ConnectProject{}, err
	}

	path := "admin/connect/projects/" + url.PathEscape(existingName)

	sh.Logger().Info("Retrieving the current Connect project", "path", path)
	currentRes, err := client.DoCustomRequest(ctx, "GET", path, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect project", err)
		return nil, ConnectProject{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for "+path+" endpoint response", "error", err)
		}
	}(currentRes)

	if err = checkResponseStatus(currentRes); err != nil {
		LogRIError(ctx, sh, "unable to retrieve Connect project", err)
		return nil, ConnectProject{}, err
	}

	currentBody, err := io.ReadAll(currentRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for GET "+path+" response", "error", err, "status", currentRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectProjectToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, ConnectProject{}, err
	}

	var project ConnectProject
	decoder := json.NewDecoder(bytes.NewReader(currentBody))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&project); err != nil {
		err = fmt.Errorf("Connect project %q could not be read safely, so it was not changed (a field this tool does not know would be lost): %w", existingName, err)
		LogRIError(ctx, sh, "unable to update Connect project", err)
		return nil, ConnectProject{}, err
	}

	if input.Description != nil {
		project.Description = *input.Description
	}
	if input.AdminGroupDN != nil {
		project.AdminGroupDN = *input.AdminGroupDN
	}
	if input.OperatorGroupDN != nil {
		project.OperatorGroupDN = *input.OperatorGroupDN
	}
	if input.AuditorGroupDN != nil {
		project.AuditorGroupDN = *input.AuditorGroupDN
	}

	requestPayload, err := json.Marshal(project)
	if err != nil {
		sh.Logger().Error("unable to marshal Connect project payload", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectProjectToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, ConnectProject{}, err
	}

	sh.Logger().Info("Calling admin/connect/projects endpoint")
	projectRes, err := client.DoCustomRequest(ctx, "POST", "admin/connect/projects", bytes.NewBuffer(requestPayload))
	if err != nil {
		LogRIError(ctx, sh, "unable to update Connect project", err)
		return nil, ConnectProject{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for admin/connect/projects endpoint response", "error", err)
		}
	}(projectRes)

	if err = checkResponseStatus(projectRes); err != nil {
		LogRIError(ctx, sh, "unable to update Connect project", err)
		return nil, ConnectProject{}, err
	}

	projectBody, err := io.ReadAll(projectRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for admin/connect/projects response", "error", err, "status", projectRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectProjectToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, ConnectProject{}, err
	}

	sh.Logger().Debug("POST admin/connect/projects response body", "body", string(projectBody))

	var updated ConnectProject

	err = json.Unmarshal(projectBody, &updated)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for POST admin/connect/projects response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", updateConnectProjectToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, ConnectProject{}, err
	}

	sh.Logger().Info("Updated Connect project successfully", "id", updated.Id)

	return nil, updated, nil
}
