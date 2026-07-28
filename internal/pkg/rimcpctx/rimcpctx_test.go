// Copyright 2026, Jamf Software LLC

package rimcpctx

import (
	"context"
	"testing"
)

func TestWithContext_RoundTrip(t *testing.T) {
	c := Context{
		LicenseeID:         "tenant-123",
		TelemetryAppID:     "app-abc",
		TelemetryNamespace: "com.example",
		Version:            "v1.0.0",
		BuildNumber:        "deadbeef",
		OS:                 "darwin",
		Arch:               "arm64",
	}

	ctx := WithContext(context.Background(), c)
	got, ok := FromContext(ctx)

	if !ok {
		t.Fatal("FromContext returned false, expected true")
	}
	if got != c {
		t.Errorf("FromContext returned %+v, want %+v", got, c)
	}
}

func TestFromContext_Absent(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Error("FromContext returned true on plain background context, expected false")
	}
}

func TestWithContext_ValueSemantics(t *testing.T) {
	c := Context{Version: "v1.0.0"}
	ctx := WithContext(context.Background(), c)

	got, _ := FromContext(ctx)
	got.Version = "mutated"

	// Original value in context must be unchanged
	got2, _ := FromContext(ctx)
	if got2.Version != "v1.0.0" {
		t.Errorf("context value was mutated: got %q, want %q", got2.Version, "v1.0.0")
	}
}
