package prompts

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPasswordResetAuditPrompt(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]string
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, args map[string]string, output *mcp.GetPromptResult)
	}{
		{
			name: "template parsed successfully",
			args: map[string]string{
				"Username":   "Tom Petty",
				"Timezone":   "America/Chicago",
				"Delegation": "All User Detail",
			},
			wantErr: false,
			assertOutput: func(t *testing.T, args map[string]string, output *mcp.GetPromptResult) {
				textContent := output.Messages[0].Content.(*mcp.TextContent)
				textOutput := textContent.Text
				if !strings.Contains(textOutput, args["Username"]) {
					t.Fatalf("expected %s to be in prompt, got %s", args["Username"], textOutput)
				}
				if !strings.Contains(textOutput, args["Timezone"]) {
					t.Fatalf("expected %s to be in prompt, got %s", args["Timezone"], textOutput)
				}
				if !strings.Contains(textOutput, args["Delegation"]) {
					t.Fatalf("expected %s to be in prompt, got %s", args["Delegation"], textOutput)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := PasswordResetAuditPromptHandler(t.Context(), newReq(tt.args))

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
			}
			if tt.assertOutput != nil {
				tt.assertOutput(t, tt.args, output)
			}
		})
	}
}
