// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"strings"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

// POST /admin/connect/projects creates or updates, and behaves like this
// (observed against a live tenant):
//
//   - The project is keyed by name, not id: the server runs
//     "update connect_projects ... where name=?". A body without a name does
//     not update anything (it fails on the unique id constraint).
//   - A POST whose name matches an existing project REPLACES that project.
//     Every field missing from the body is cleared (description, groups,
//     restPoints), and the project is given a new id unless the body carries
//     the existing one. There is no "already exists" error.
//
// So the server never protects a project from being overwritten:
// create-connect-project refuses a name that is already in use, and
// update-connect-project refuses a name no project has and sends back the
// whole current project, with its id, changing only the requested fields.

// ConnectProject is a Connect project as GET and POST /admin/connect/projects
// exchange it. It is defined here instead of using rapididentity.ConnectProject
// because the SDK type (v1.7.0) lacks fields the API returns inside REST
// points: a REST point's consumes and its own authSpec, and an argMap entry's
// sourceKey. Since a POST replaces the whole project, posting the SDK type back
// would silently clear them; on one tenant 23 of 30 projects had such REST
// points. Fields the API sometimes leaves out are pointers, so that a round
// trip sends back exactly what it received.
type ConnectProject struct {
	Name              string                  `json:"name" jsonschema:"The name of the project."`
	Id                string                  `json:"id" jsonschema:"The unique ID of the project."`
	Description       string                  `json:"description" jsonschema:"The description of the project."`
	AdminGroupDN      string                  `json:"adminGroupDN" jsonschema:"The DN of the group with administrator privileges for the project."`
	OperatorGroupDN   string                  `json:"operatorGroupDN" jsonschema:"The DN of the group with operator privileges for the project."`
	AuditorGroupDN    string                  `json:"auditorGroupDN" jsonschema:"The DN of the group with auditor privileges for the project."`
	AdminGroupName    *string                 `json:"adminGroupName,omitempty" jsonschema:"The display name of the admin group, when the server provides it."`
	OperatorGroupName *string                 `json:"operatorGroupName,omitempty" jsonschema:"The display name of the operator group, when the server provides it."`
	AuditorGroupName  *string                 `json:"auditorGroupName,omitempty" jsonschema:"The display name of the auditor group, when the server provides it."`
	RestPoints        *ConnectRestPointConfig `json:"restPoints,omitempty" jsonschema:"The project's RESTPoints."`
	HttpStatus        *int                    `json:"httpStatus,omitempty" jsonschema:"The HTTP status, when the server includes it."`
	ChangeCount       int                     `json:"changeCount" jsonschema:"The server's change counter."`
	ModifiedMs        int64                   `json:"modifiedMs" jsonschema:"When the project was last modified, in milliseconds since the epoch."`
	ModifiedBy        string                  `json:"modifiedBy" jsonschema:"The ID of who last modified the project."`
	ModifiedByName    string                  `json:"modifiedByName" jsonschema:"The name of who last modified the project."`
}

// ConnectRestPointConfig is a project's RESTPoint configuration.
type ConnectRestPointConfig struct {
	Disabled   bool                `json:"disabled" jsonschema:"Whether the project's RESTPoints are disabled."`
	AuthSpec   ConnectAuthSpec     `json:"authSpec" jsonschema:"The default authentication for the RESTPoints."`
	RestPoints *[]ConnectRestPoint `json:"restPoints,omitempty" jsonschema:"The RESTPoints."`
}

// ConnectRestPoint exposes an action set as a REST endpoint.
type ConnectRestPoint struct {
	Id          string               `json:"id" jsonschema:"The unique ID of the RESTPoint."`
	Description string               `json:"description" jsonschema:"The description of the RESTPoint."`
	Method      string               `json:"method" jsonschema:"The HTTP method."`
	Disabled    bool                 `json:"disabled" jsonschema:"Whether the RESTPoint is disabled."`
	Path        string               `json:"path" jsonschema:"The HTTP path."`
	Consumes    *string              `json:"consumes,omitempty" jsonschema:"The Content-Type the RESTPoint accepts."`
	Produces    string               `json:"produces" jsonschema:"The Content-Type the RESTPoint produces."`
	ActionSet   string               `json:"actionSet" jsonschema:"The action set the RESTPoint calls."`
	ArgMap      *[]ConnectArgMapItem `json:"argMap,omitempty" jsonschema:"How request data maps to the action set's arguments."`
	AuthSpec    *ConnectAuthSpec     `json:"authSpec,omitempty" jsonschema:"The RESTPoint's own authentication, overriding the project default."`
}

// ConnectArgMapItem maps one piece of request data to an action set argument.
type ConnectArgMapItem struct {
	SourceKey  *string `json:"sourceKey,omitempty" jsonschema:"The request key to read."`
	SourceType *string `json:"sourceType,omitempty" jsonschema:"Where in the request the value comes from."`
	DestKey    string  `json:"destKey" jsonschema:"The action set argument to fill."`
	DestType   string  `json:"destType" jsonschema:"The argument's type."`
}

// ConnectAuthSpec lists the authentication methods a RESTPoint accepts.
type ConnectAuthSpec struct {
	Anonymous          bool `json:"anonymous" jsonschema:"Whether anonymous access is allowed."`
	Oauth1             bool `json:"oauth1" jsonschema:"Whether OAuth 1 is accepted."`
	Basic              bool `json:"basic" jsonschema:"Whether basic authentication is accepted."`
	BasicWithOAuthKeys bool `json:"basicWithOAuthKeys" jsonschema:"Whether basic authentication with OAuth keys is accepted."`
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
