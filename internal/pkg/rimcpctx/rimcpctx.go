// Copyright 2026, Jamf Software LLC

package rimcpctx

import "context"

// Context holds static server metadata known at startup.
// Injected into the root context in main.go and propagated to all handlers.
// Telemetry-specific fields (TelemetryAppID, TelemetryNamespace, LicenseeID)
// are only populated when MCP_RAPIDID_TELEMETRY=true.
type Context struct {
	LicenseeID         string
	TelemetryAppID     string // TelemetryDeck app ID — baked via ldflags
	TelemetryNamespace string // TelemetryDeck namespace — baked via ldflags
	Version            string
	BuildNumber        string // commit SHA
	OS                 string // runtime.GOOS — captured at startup
	Arch               string // runtime.GOARCH — captured at startup
}

type contextKey struct{}

// WithContext returns a new context carrying c as a value.
// c is stored by value to prevent cross-goroutine mutation.
func WithContext(ctx context.Context, c Context) context.Context {
	return context.WithValue(ctx, contextKey{}, c)
}

// FromContext retrieves the Context value from ctx.
// Returns the zero value and false if no Context was injected.
func FromContext(ctx context.Context) (Context, bool) {
	c, ok := ctx.Value(contextKey{}).(Context)
	return c, ok
}
