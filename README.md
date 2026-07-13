# RapidID MCP Server

[![mcp-rapidid release (latest SemVer)](https://img.shields.io/github/v/release/Jamf-Concepts/mcp-rapidid?sort=semver)](https://github.com/Jamf-Concepts/mcp-rapidid/releases)

The RapidID MCP Server enables AI applications that support the
Model Context Protocol (MCP) — such as Claude Desktop — to interact with RapidID
identity management systems. It exposes tools for
searching users, managing entitlements, querying groups, reviewing audit logs, and working
with Connect action sets.

## Getting Started

### Using Claude Extensions

1. Download mcpb package for your OS
2. Open Claude Desktop and go to Settings ➡️ Extensions ➡️ Advanced Settings ➡️ Install Extension
3. Choose mcpb package from step 1
4. Enter Hostname and Username/Password OR Service Identity

### Using Binary

Download the binary for your OS from releases and
follow the Usage with Model Context Protocol instructions.

## Usage with Model Context Protocol

### Best practices for credentials

Do not use your personal RapidIdentity account as the MCP server’s credentials. Prefer a **sponsored identity** dedicated to this integration: stewardship keeps ownership clear, actions show up under that identity in the audit trail, and you can adjust group membership or reset credentials without touching anyone’s day-to-day login.

Some tools require a **Service Identity** (see the table below). When you use one, grant only the **least-privileged** RapidIdentity groups needed for the tools you enable—typically a single legend entry such as **[2]** for workflow entitlements—rather than Tenant Admin “just in case.”

To integrate this server with apps that support MCP using your
RapidIdentity username and password

```json
{
  "mcpServers": {
    "mcp-rapidid": {
      "command": "<path to downloaded binary>",
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
service identities. Service Identities can call many endpoints when
granted the right RapidIdentity groups (see the table below), but some tools
still require a user session (username/password) because they depend on
delegations or other user-scoped APIs.

```json
{
  "mcpServers": {
    "mcp-rapidid": {
      "command": "<path to downloaded binary>",
      "env": {
        "RI_SERVICE_IDENTITY_SECRET_KEY": "1jdie203i4jjf9",
        "RI_HOST": "portal.us006-rapididentity.com"
      }
    }
  }
}
```

## Tools

The "Service ID Compatible" column indicates whether a RapidID Service Identity can call the underlying API endpoint. Where compatible, the Service Identity must be a member of at least one of the groups listed in the legend below the table.

| Tool | Description | Service ID Compatible |
|------|-------------|----------------------|
| `search-users` | Performs a simple search based on the delegations available to the authenticated user | No |
| `search-entitlements-for-user` | Performs a search of entitlements for the given user based on their idautoID | Yes [2] |
| `start-entitlement-request` | Initiates an entitlement request for a particular user and entitlement based on idautoID and resourceId respectively | Yes [2] |
| `get-my-delegations` | Gets delegations that are accessible to the authenticated user | No |
| `get-user-info-in-delegation` | Does an advanced search of a RapidID delegation | No |
| `search-groups` | Does a simple search of a RapidID group | Yes [3] |
| `get-group-members` | Gets group members for a specified RapidID group | Yes [4] |
| `get-user-activity-from-audit-log` | Returns audit log activity for a specific RapidID user over a given date range | Yes [1] |
| `get-connect-projects` | Returns all RapidID Connect projects | Yes [5] |
| `get-connect-actions` | Returns Connect action sets within a project, or across all projects | Yes [5] |
| `get-connect-action` | Returns a single RapidID Connect action set by ID | Yes [5] |
| `save-connect-action` | Saves (creates or updates) a RapidID Connect action set | Yes [6] |
| `delete-connect-action` | Deletes a RapidID Connect action set by ID | Yes [5] |
| `get-password-policies-for` | Retrieves the password policy for specified users | No |
| `set-password` | Sets the RapidID password for one or more users via delegations | No |
| `run-connect-action` | Runs a RapidID Connect action set and returns the HTML log | Yes [6] |
| `get-connect-files` | Returns metadata for files and directories within the RapidID Connect files module | Yes [6] |
| `get-connect-file-content` | Returns the text content of a file from the RapidID Connect files module | Yes [6] |

**Service ID Group Legend**

| # | Groups |
|---|--------|
| [1] | System Admin, Tenant Admin, Reporting Admin, Reporting Viewer, District Admin, District Manager |
| [2] | System Admin, Tenant Admin, Workflow Admin, Workflow Helpdesk |
| [3] | System Admin, Tenant Admin, Groups Module Admin, Groups Module Helpdesk, Groups Module Viewer, District Manager |
| [4] | System Admin, Tenant Admin, Groups Module Admin, Groups Module Viewer |
| [5] | Tenant Admin, Connect Admin, Reporting Admin, Reporting Viewer, Workflow Admin, Workflow Helpdesk, Groups Module Admin, Groups Module Helpdesk, Groups Module Viewer, District Admin, District Manager |
| [6] | Tenant Admin, Connect Admin |

Legend **[2]** covers the workflow entitlement tools (`search-entitlements-for-user`, `start-entitlement-request`). Membership in **any one** of those groups is enough. Roles outside that set — including Reporting Admin/Viewer, Connect Admin, Groups Module Admin/Helpdesk/Viewer, and District Admin/Manager — return HTTP 403 on those endpoints even though they unlock other tools via legends [1], [3], [4], [5], or [6].

## Skills

- [RapidID Role Mining](./skills/rapididentity-role-mining/SKILL.md): Process for identifying dynamic filters for static RapidID groups.
- [Connect Action Sets](./skills/connect-action-sets/SKILL.md): Knowledge on how to work with RapidID Connect action sets.

## Troubleshooting

- On authentication failures ensure RI_HOST, RI_USER / RI_PASSWORD, or RI_SERVICE_IDENTITY_SECRET_KEY are set correctly
- If you receive an unexpected empty array `[]` when using the `search-users` or `get-user-info-in-delegation`, this is most likely due to utilizing service identities and switching to username and password will resolve the issue (those tools are not Service ID compatible)
- Service Identities that lack the groups listed for a tool typically receive HTTP 403 with `Insufficient Authorization`. Tools marked **No** either reject Service Identities outright or depend on user-session delegations
- The `get-user-info-in-delegation` does not support pagination, which can cause tool response size errors. A workaround for this is to use a combination of the `search-groups` and the `get-group-members` tools to chunk out users into multiple tool calls as the `get-group-members` tool supports pagination.
- If you receive a save error when using the `save-connect-actions` tool this is most likely due to not iterating the version number. Ensure that the most recent version of the action set is retrieved first using `get-connect-action` so that you iterate the version number properly
- The `get-user-info-in-delegation` takes a raw LDAP filter input. In circumstances where no results are returned this could be caused by a malformed LDAP filter.

## Getting Help

Open an issue at https://github.com/Jamf-Concepts/mcp-rapidid/issues
