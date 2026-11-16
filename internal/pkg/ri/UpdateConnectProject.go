// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"

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
// because the server would otherwise create a new project; see
// saveConnectProject.
func UpdateConnectProject(ctx context.Context, req *mcp.CallToolRequest, input UpdateConnectProjectInput) (*mcp.CallToolResult, rapididentity.ConnectProject, error) {
	changes := map[string]any{}
	for field, value := range map[string]*string{
		"description":     input.Description,
		"adminGroupDN":    input.AdminGroupDN,
		"operatorGroupDN": input.OperatorGroupDN,
		"auditorGroupDN":  input.AuditorGroupDN,
	} {
		if value != nil {
			changes[field] = *value
		}
	}
	return saveConnectProject(ctx, req, updateConnectProjectToolName, input.Name, true, changes)
}
