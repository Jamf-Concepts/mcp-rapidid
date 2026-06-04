package main

import (
	"context"
	"log"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/ri"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "mcp-rapidid", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "search-users", Description: "Used to search basic Rapididentity user information based on a simple criteria"}, ri.SearchRapidIdentityUsers)
	mcp.AddTool(server, &mcp.Tool{Name: "search-entitlements-for-user", Description: "Used to search entitlements for a RapidIdentity user"}, ri.GetEntitlementForUser)
	mcp.AddTool(server, &mcp.Tool{Name: "start-entitlement-request", Description: "Starts entitlement requests for any number of users"}, ri.StartEntitlementRequest)
	mcp.AddTool(server, &mcp.Tool{Name: "get-my-delegations", Description: "Returns the delegations accessible to the authenticated user"}, ri.GetMyDelegations)
	mcp.AddTool(server, &mcp.Tool{Name: "get-user-info-in-delegation", Description: "Returns user profile information for users in a specific RapidIdentity delegation. Provides a more complete view of the user beyond basic user information"}, ri.GetUserInfoInDelegation)
	mcp.AddTool(server, &mcp.Tool{Name: "search-groups", Description: "Searches for RapidIdentity groups matching the given criteria"}, ri.SearchGroups)
	mcp.AddTool(server, &mcp.Tool{Name: "get-group-members", Description: "Returns the members of a specific RapidIdentity group"}, ri.GetGroupMembers)
	mcp.AddTool(server, &mcp.Tool{Name: "get-user-activity-from-audit-log", Description: "Returns audit log activity for a specific RapidIdentity user over a given date range"}, ri.GetUserActivityFromAuditLog)
	mcp.AddTool(server, &mcp.Tool{Name: "get-connect-projects", Description: "Returns all RapidIdentity Connect projects"}, ri.GetConnectProjects)
	mcp.AddTool(server, &mcp.Tool{Name: "get-connect-actions", Description: "Returns Connect action sets within a project, or across all projects", OutputSchema: ri.ConnectActionsOutputSchema}, ri.GetConnectActions)
	mcp.AddTool(server, &mcp.Tool{Name: "get-connect-action", Description: "Returns a single RapidIdentity Connect action set by ID", OutputSchema: ri.ConnectActionDefSchema}, ri.GetConnectAction)
	mcp.AddTool(server, &mcp.Tool{Name: "save-connect-action", Description: "Saves (creates or updates) a RapidIdentity Connect action set", InputSchema: ri.SaveConnectActionInputSchema, OutputSchema: ri.ConnectActionDefSchema}, ri.SaveConnectAction)
	mcp.AddTool(server, &mcp.Tool{Name: "delete-connect-action", Description: "Deletes a RapidIdentity Connect action set by ID"}, ri.DeleteConnectAction)
	err := server.Run(context.Background(), &mcp.StdioTransport{})
	if err != nil {
		log.Fatal(err)
	}
}
