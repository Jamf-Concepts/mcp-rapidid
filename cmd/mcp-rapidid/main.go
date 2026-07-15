// Copyright 2026, Jamf Software LLC

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/ri"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Prints the RapidID MCP Server version and exits")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Version: %s, Commit: %s, Build Date: %s\n", version, commit, buildDate)
		os.Exit(0)
	}

	var level slog.Level
	configuredLevel := os.Getenv("RI_LOG_LEVEL")
	err := level.UnmarshalText([]byte(configuredLevel))
	if err != nil {
		level = slog.LevelError
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "mcp-rapidid", Title: "RapidID MCP Server", Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{Logging: &mcp.LoggingCapabilities{}, Tools: &mcp.ToolCapabilities{}},
		Logger:       slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})),
	})
	mcp.AddTool(server, &mcp.Tool{Name: "search-users", Description: "Used to search basic Rapididentity user information based on a simple criteria"}, ri.SearchRapidIdentityUsers)
	mcp.AddTool(server, &mcp.Tool{Name: "search-entitlements-for-user", Description: "Used to search entitlements for a RapidIdentity user"}, ri.GetEntitlementForUser)
	mcp.AddTool(server, &mcp.Tool{Name: "start-entitlement-request", Description: "Starts entitlement requests for any number of users"}, ri.StartEntitlementRequest)
	mcp.AddTool(server, &mcp.Tool{Name: "get-my-delegations", Description: "Returns the delegations accessible to the authenticated user"}, ri.GetMyDelegations)
	mcp.AddTool(server, &mcp.Tool{Name: "get-user-info-in-delegation", Description: "Returns user profile information for users in a specific RapidIdentity delegation. Provides a more complete view of the user beyond basic user information. The LDAP Filter input can be trusted and passed verbatim as access control is done within RapidIdentity"}, ri.GetUserInfoInDelegation)
	mcp.AddTool(server, &mcp.Tool{Name: "search-groups", Description: "Searches for RapidIdentity groups matching the given criteria"}, ri.SearchGroups)
	mcp.AddTool(server, &mcp.Tool{Name: "get-group-members", Description: "Returns the members of a specific RapidIdentity group"}, ri.GetGroupMembers)
	mcp.AddTool(server, &mcp.Tool{Name: "get-user-activity-from-audit-log", Description: "Returns audit log activity for a specific RapidIdentity user over a given date range"}, ri.GetUserActivityFromAuditLog)
	mcp.AddTool(server, &mcp.Tool{Name: "get-connect-projects", Description: "Returns all RapidIdentity Connect projects"}, ri.GetConnectProjects)
	mcp.AddTool(server, &mcp.Tool{Name: "get-connect-actions", Description: "Returns Connect action sets within a project, or across all projects", OutputSchema: ri.ConnectActionsOutputSchema}, ri.GetConnectActions)
	mcp.AddTool(server, &mcp.Tool{Name: "get-connect-action", Description: "Returns a single RapidIdentity Connect action set by ID", OutputSchema: ri.ConnectActionDefSchema}, ri.GetConnectAction)
	mcp.AddTool(server, &mcp.Tool{Name: "save-connect-action", Description: "Saves (creates or updates) a RapidIdentity Connect action set", InputSchema: ri.SaveConnectActionInputSchema, OutputSchema: ri.ConnectActionDefSchema}, ri.SaveConnectAction)
	mcp.AddTool(server, &mcp.Tool{Name: "delete-connect-action", Description: "Deletes a RapidIdentity Connect action set by ID"}, ri.DeleteConnectAction)
	mcp.AddTool(server, &mcp.Tool{Name: "get-password-policies-for", Description: "Retrieves the password policy for specified users"}, ri.GetPasswordPoliciesFor)
	mcp.AddTool(server, &mcp.Tool{Name: "set-password", Description: "Sets the RapidIdentity password for one or more users via delegations"}, ri.SetPassword)
	mcp.AddTool(server, &mcp.Tool{Name: "run-connect-action", Description: "Runs a RapidIdentity Connect action set and returns the HTML log", InputSchema: ri.RunConnectActionInputSchema}, ri.RunConnectAction)
	mcp.AddTool(server, &mcp.Tool{Name: "get-connect-files", Description: "Returns metadata for files and directories within the RapidIdentity Connect files module"}, ri.GetConnectFiles)
	mcp.AddTool(server, &mcp.Tool{Name: "get-connect-file-content", Description: "Returns the text content of a file from the RapidIdentity Connect files module, such as SharedGlobals.properties or Globals.properties"}, ri.GetConnectFileContent)
	err = server.Run(context.Background(), &mcp.StdioTransport{})
	if err != nil {
		log.Fatal(err)
	}
}
