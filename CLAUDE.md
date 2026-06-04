# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the **RapidIdentity MCP Server** — a Go-based Model Context Protocol (MCP) server that bridges MCP-compatible AI applications (like Claude) with RapidIdentity identity management systems. It exposes three tools for searching users, retrieving entitlements, and initiating entitlement requests.

## Commands

```bash
# Build
go build ./cmd/mcp-rapidid

# Test
go test ./...

# Format (required before committing)
./scripts/fmt.sh
```

## Architecture

**Entry point:** `cmd/mcp-rapidid/main.go` — initializes the MCP server and registers the three tools.

**Tools** (`internal/pkg/ri/`, one file per tool):
- `SearchUsers.go` — search users by text criteria across delegations
- `GetEntitlementsForUser.go` — retrieve entitlements and resource associations for a user
- `StartEntitlementRequest.go` — initiate GRANT/REVOKE entitlement requests

**Client config:** `internal/pkg/ri/ri.go` — `GetRapidIdentityOptions()` reads environment variables and returns authenticated client options. Service identity auth takes precedence over username/password.

**Key dependencies:**
- `github.com/modelcontextprotocol/go-sdk` — MCP protocol
- `github.com/hatch-ed-com/ri-sdk-go` — RapidIdentity API client (uses `DoCustomRequest()` for HTTP calls)

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `RI_HOST` | RapidIdentity hostname (always required) |
| `RI_USER` | Username (for user/password auth) |
| `RI_PASSWORD` | Password (for user/password auth) |
| `RI_SERVICE_IDENTITY_SECRET_KEY` | Service identity key (takes precedence over user/password) |

## Development Conventions

- Each MCP tool lives in its own `.go` file under `internal/pkg/ri/`
- New tools must be registered in `main.go`
- Run `./scripts/fmt.sh` before every commit
- No force-push; the repo uses squash-and-merge

## MCPB Bundle

The `build/package/mcpb/` directory packages this server as an MCPB (MCP Bundle) following the [Go standard project layout](https://github.com/golang-standards/project-layout).

- `build/package/mcpb/manifest.json` — MCPB manifest v0.3: metadata, `user_config` (maps RI_* env vars), and all 8 tool declarations
- `build/package/mcpb/server/` — compiled binary destination (git-ignored; populated by build script)
- `build/package/mcpb/.mcpbignore` — excludes source files from `mcpb pack`

### Building the Bundle

```bash
npm install -g @anthropic-ai/mcpb          # one-time
./scripts/build-bundle.sh                  # compiles binary into build/package/mcpb/server/
cd build/package/mcpb/
mcpb validate
mcpb pack
```

Cross-compile for all platforms: `./scripts/build-bundle.sh --all-platforms`

### user_config → Env var mapping

| user_config key | Env var | Sensitive | Required |
|---|---|---|---|
| `ri_host` | `RI_HOST` | no | yes |
| `ri_user` | `RI_USER` | no | no |
| `ri_password` | `RI_PASSWORD` | yes | no |
| `ri_service_identity_secret_key` | `RI_SERVICE_IDENTITY_SECRET_KEY` | yes | no |
