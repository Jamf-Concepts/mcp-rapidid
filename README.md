# RapidIdentity MCP Server

[![mcp-rapidid release (latest SemVer)](https://img.shields.io/github/v/release/Jamf-Concepts/mcp-rapidid?sort=semver)](https://github.com/Jamf-Concepts/mcp-rapidid/releases)

## Getting Started

### Using Claude Extensions

1. Download mcpb package for your OS
2. Open Claude Desktop and go to Settings ➡️ Extensions ➡️ Advanced Settings ➡️ Install Extension
3. Choose mcpb package from step 1
4. Enter Hostname and Username/Password OR Service Identity

### Using Go Install

```
go install github.com/Jamf-Concepts/mcp-rapidid/cmd/mcp-rapidid
```

This will install the binary in the bin folder within your GOPATH. The GOPATH environment
variable can be found by running the following command `go env`. Typically, the path to
the binary, once installed, on macOS will be `/Users/<username>/go/bin/mcp-rapidid`.

## Usage with Model Context Protocol

To integrate this server with apps that support MCP using your
RapidIdentity username and password

```json
{
  "mcpServers": {
    "mcp-rapidid": {
      "command": "/Users/kelly/go/bin/mcp-rapidid",
      "env": {
        "RI_USER": "kclarkson",
        "RI_PASSWORD": "notarealpassword123",
        "RI_HOST": "portal.us006-rapididentity.com"
      }
    }
  }
}
```

To integrate this server with apps that support MCP using RapidIdentity
service identities. Keep in mind that Service Identities do not have
access to all API endpoints even with the Tenant Admin role.

```json
{
  "mcpServers": {
    "mcp-rapidid": {
      "command": "/Users/kelly/go/bin/mcp-rapidid",
      "env": {
        "RI_SERVICE_IDENTITY_SECRET_KEY": "1jdie203i4jjf9",
        "RI_HOST": "portal.us006-rapididentity.com"
      }
    }
  }
}
```

## Tools

- `search-users`: Performs a simple search based on the delegations available to the authenticated user.
- `search-entitlements-for-user`: Performs a search of entitlements for the given user based on their idautoID.
- `start-entitlement-request`: Initiates an entitlement request for a particular user and entitlement based on idautoID and resourceId respectively.
- `get-my-delegations`: Gets delegations that are accessible to the authenticated user. This is based on the credentials included in your environment variables.
- `get-user-info-in-delegation`: Does an advanced search of a RapidID delegation.
- `search-groups`: Does a simple search of a RapidID group.
- `get-group-members`: Gets group members for a specificed RapidID group.
- `get-user-activity-from-audit-log`: Returns audit log activity for a specific RapidIdentity user over a given date range.
- `get-connect-projects`: Returns all RapidIdentity Connect projects
- `get-connect-actions`: Returns Connect action sets within a project, or across all projects
- `get-connect-action`: Returns a single RapidIdentity Connect action set by ID
- `save-connect-action`: Saves (creates or updates) a RapidIdentity Connect action set
- `delete-connect-action`: Deletes a RapidIdentity Connect action set by ID

## Skills

- [RapidIdentity Role Mining](./skills/rapididentity-role-mining/SKILL.md): Process for identifying dynamic filters for static RapidID groups.
- [Connect Action Sets](./skills/connect-action-sets/SKILL.md): Knowledge on how to work with RapidID Connect action sets.
