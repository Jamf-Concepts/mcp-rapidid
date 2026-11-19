// Copyright 2026, Jamf Software LLC

package ri

import (
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

// TestOutputSchemasAreObjects guards the rule MCP clients enforce: a tool's
// output schema must have "type": "object". Claude Desktop rejects the whole
// tool list, not just the one tool, when any output schema breaks it.
func TestOutputSchemasAreObjects(t *testing.T) {
	schemas := map[string]struct{ typ string }{
		"ConnectActionDefSchema":     {ConnectActionDefSchema.Type},
		"ConnectActionsOutputSchema": {ConnectActionsOutputSchema.Type},
		"ConnectJobSchema":           {ConnectJobSchema.Type},
		"ConnectJobsOutputSchema":    {ConnectJobsOutputSchema.Type},
	}
	for name, s := range schemas {
		if s.typ != "object" {
			t.Errorf("%s has type %q, want \"object\"", name, s.typ)
		}
	}
	if ConnectJobSchema.Ref != "" {
		t.Errorf("ConnectJobSchema must not be a bare $ref, got %q", ConnectJobSchema.Ref)
	}
	if _, ok := ConnectJobSchema.Properties["action"]; !ok {
		t.Error("ConnectJobSchema is missing the job's properties")
	}
}

// TestSchemasResolve resolves every hand-written schema the way the go-sdk
// does when a tool is registered. A schema that does not resolve, such as
// one whose nodes do not form a tree, makes AddTool panic at startup, which
// the tool tests alone would never show.
func TestSchemasResolve(t *testing.T) {
	schemas := map[string]*jsonschema.Schema{
		"ConnectActionDefSchema":       ConnectActionDefSchema,
		"ConnectActionsOutputSchema":   ConnectActionsOutputSchema,
		"SaveConnectActionInputSchema": SaveConnectActionInputSchema,
		"RunConnectActionInputSchema":  RunConnectActionInputSchema,
		"ConnectJobSchema":             ConnectJobSchema,
		"ConnectJobsOutputSchema":      ConnectJobsOutputSchema,
	}
	for name, schema := range schemas {
		if _, err := schema.Resolve(nil); err != nil {
			t.Errorf("%s does not resolve: %v", name, err)
		}
	}
}
