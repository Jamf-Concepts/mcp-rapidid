// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const createConnectProjectToolName = "create-connect-project"

type CreateConnectProjectInput struct {
	Name            string `json:"name" jsonschema:"The name of the new Connect project"`
	AdminGroupDN    string `json:"adminGroupDN,omitempty" jsonschema:"The LDAP distinguished name (DN) of the project's admin group"`
	OperatorGroupDN string `json:"operatorGroupDN,omitempty" jsonschema:"The LDAP distinguished name (DN) of the project's operator group"`
	AuditorGroupDN  string `json:"auditorGroupDN,omitempty" jsonschema:"The LDAP distinguished name (DN) of the project's auditor group"`
}

// createConnectProjectBody is the body for a new project: a name and the
// groups given, and no id, so that the server creates the project.
type createConnectProjectBody struct {
	Name            string `json:"name"`
	AdminGroupDN    string `json:"adminGroupDN,omitempty"`
	OperatorGroupDN string `json:"operatorGroupDN,omitempty"`
	AuditorGroupDN  string `json:"auditorGroupDN,omitempty"`
}

// CreateConnectProject creates a project with a name and, optionally, its
// admin, operator and auditor groups. It refuses a name that is already in
// use, ignoring case, because the server would otherwise replace that project
// and clear its fields; see ConnectProjects.go.
func CreateConnectProject(ctx context.Context, req *mcp.CallToolRequest, input CreateConnectProjectInput) (*mcp.CallToolResult, ConnectProject, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ConnectProject{}, fmt.Errorf("a project name is required")
	}

	var err error
	ctx, client, sh, setupErr := ToolSetup(ctx, req, createConnectProjectToolName)
	if setupErr != nil {
		return nil, ConnectProject{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, createConnectProjectToolName, start, &err)

	sh.Logger().Info(createConnectProjectToolName+" tool called", "name", name)

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	existingName, err := findConnectProjectName(ctx, client, sh, name)
	if err != nil {
		return nil, ConnectProject{}, err
	}
	if existingName != "" {
		err = fmt.Errorf("a Connect project named %q already exists", existingName)
		LogRIError(ctx, sh, "unable to create Connect project", err)
		return nil, ConnectProject{}, err
	}

	requestPayload, err := json.Marshal(createConnectProjectBody{
		Name:            name,
		AdminGroupDN:    input.AdminGroupDN,
		OperatorGroupDN: input.OperatorGroupDN,
		AuditorGroupDN:  input.AuditorGroupDN,
	})
	if err != nil {
		sh.Logger().Error("unable to marshal Connect project payload", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectProjectToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, ConnectProject{}, err
	}

	sh.Logger().Info("Calling admin/connect/projects endpoint")
	projectRes, err := client.DoCustomRequest(ctx, "POST", "admin/connect/projects", bytes.NewBuffer(requestPayload))
	if err != nil {
		LogRIError(ctx, sh, "unable to create Connect project", err)
		return nil, ConnectProject{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for admin/connect/projects endpoint response", "error", err)
		}
	}(projectRes)

	if err = checkResponseStatus(projectRes); err != nil {
		LogRIError(ctx, sh, "unable to create Connect project", err)
		return nil, ConnectProject{}, err
	}

	projectBody, err := io.ReadAll(projectRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for admin/connect/projects response", "error", err, "status", projectRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectProjectToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, ConnectProject{}, err
	}

	sh.Logger().Debug("POST admin/connect/projects response body", "body", string(projectBody))

	var project ConnectProject

	err = json.Unmarshal(projectBody, &project)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for POST admin/connect/projects response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectProjectToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, ConnectProject{}, err
	}

	sh.Logger().Info("Created Connect project successfully", "id", project.Id)

	return nil, project, nil
}
