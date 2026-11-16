// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

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

// CreateConnectProject creates a project with a name and, optionally, its
// admin, operator and auditor groups. It refuses a name that is already in
// use, because the server would otherwise replace that project and clear its
// fields; see saveConnectProject.
func CreateConnectProject(ctx context.Context, req *mcp.CallToolRequest, input CreateConnectProjectInput) (*mcp.CallToolResult, rapididentity.ConnectProject, error) {
	changes := map[string]any{}
	for field, value := range map[string]string{
		"adminGroupDN":    input.AdminGroupDN,
		"operatorGroupDN": input.OperatorGroupDN,
		"auditorGroupDN":  input.AuditorGroupDN,
	} {
		if value != "" {
			changes[field] = value
		}
	}
	return saveConnectProject(ctx, req, createConnectProjectToolName, input.Name, false, changes)
}
