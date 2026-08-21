package prompts

import (
	"context"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/ri"
	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func PromptSetup(ctx context.Context, req *mcp.GetPromptRequest, promptName string) (context.Context, *helper.ServerHelper, error) {
	clientName, clientVersion := ri.ClientInfoFromSession(req.Session)
	// StdioTransport does not assign session IDs — req.Session.ID() always returns "".
	// The SDK's GetSessionID option only applies to HTTP/SSE transports.
	// Passing "" here is intentional; if the SDK ever starts providing session IDs
	// for stdio, swapping this to req.Session.ID() will automatically enable grouping.
	ctx = telemetry.WithSession(ctx, req.Session.ID(), clientName, clientVersion)

	sh := helper.NewPromptServerHelper(req, promptName)

	return ctx, sh, nil
}
