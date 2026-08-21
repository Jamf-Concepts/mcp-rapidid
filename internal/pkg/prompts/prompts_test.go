package prompts

import "github.com/modelcontextprotocol/go-sdk/mcp"

func newReq(args map[string]string) *mcp.GetPromptRequest {
	return &mcp.GetPromptRequest{
		Session: &mcp.ServerSession{},
		Params: &mcp.GetPromptParams{
			Arguments: args,
		},
	}
}
