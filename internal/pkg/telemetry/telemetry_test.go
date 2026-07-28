// Copyright 2026, Jamf Software LLC

package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/rimcpctx"
)

func fullCtx(version string) context.Context {
	ctx := rimcpctx.WithContext(context.Background(), rimcpctx.Context{
		LicenseeID:         "tenant-123",
		TelemetryAppID:     "app-abc",
		TelemetryNamespace: "com.example",
		Version:            version,
		BuildNumber:        "deadbeef",
		OS:                 "darwin",
		Arch:               "arm64",
	})
	return WithSession(ctx, "", "claude-code", "1.0.0")
}

func TestCanSend_MissingRimcpctx(t *testing.T) {
	ctx := WithSession(context.Background(), "", "claude-code", "1.0.0")
	_, _, ok := canSend(ctx)
	if ok {
		t.Error("canSend should return false when rimcpctx is absent")
	}
}

func TestCanSend_MissingSession(t *testing.T) {
	ctx := rimcpctx.WithContext(context.Background(), rimcpctx.Context{
		LicenseeID:         "tenant-123",
		TelemetryAppID:     "app-abc",
		TelemetryNamespace: "com.example",
		Version:            "v1.0.0",
	})
	_, _, ok := canSend(ctx)
	if ok {
		t.Error("canSend should return false when session is absent")
	}
}

func TestCanSend_EmptySessionID(t *testing.T) {
	ctx := rimcpctx.WithContext(context.Background(), rimcpctx.Context{
		LicenseeID:         "tenant-123",
		TelemetryAppID:     "app-abc",
		TelemetryNamespace: "com.example",
		Version:            "v1.0.0",
	})
	ctx = WithSession(ctx, "", "claude-code", "1.0.0")
	_, _, ok := canSend(ctx)
	if !ok {
		t.Error("canSend should return true when sessionID is empty — StdioTransport never provides one")
	}
}

func TestCanSend_EmptyLicenseeID(t *testing.T) {
	ctx := rimcpctx.WithContext(context.Background(), rimcpctx.Context{
		LicenseeID:         "",
		TelemetryAppID:     "app-abc",
		TelemetryNamespace: "com.example",
		Version:            "v1.0.0",
	})
	ctx = WithSession(ctx, "", "claude-code", "1.0.0")
	_, _, ok := canSend(ctx)
	if ok {
		t.Error("canSend should return false when LicenseeID is empty")
	}
}

func TestCanSend_EmptyAppID(t *testing.T) {
	ctx := rimcpctx.WithContext(context.Background(), rimcpctx.Context{
		LicenseeID:         "tenant-123",
		TelemetryAppID:     "",
		TelemetryNamespace: "com.example",
		Version:            "v1.0.0",
	})
	ctx = WithSession(ctx, "", "claude-code", "1.0.0")
	_, _, ok := canSend(ctx)
	if ok {
		t.Error("canSend should return false when TelemetryAppID is empty")
	}
}

func TestCanSend_EmptyNamespace(t *testing.T) {
	ctx := rimcpctx.WithContext(context.Background(), rimcpctx.Context{
		LicenseeID:         "tenant-123",
		TelemetryAppID:     "app-abc",
		TelemetryNamespace: "",
		Version:            "v1.0.0",
	})
	ctx = WithSession(ctx, "", "claude-code", "1.0.0")
	_, _, ok := canSend(ctx)
	if ok {
		t.Error("canSend should return false when TelemetryNamespace is empty")
	}
}

func TestCanSend_AllPresent(t *testing.T) {
	_, _, ok := canSend(fullCtx("v1.0.0"))
	if !ok {
		t.Error("canSend should return true when all prerequisites are present")
	}
}

func TestIsTestMode_DevVersion(t *testing.T) {
	rctx, _, ok := canSend(fullCtx("dev"))
	if !ok {
		t.Fatal("canSend returned false unexpectedly")
	}
	ev := event{IsTestMode: rctx.Version == "dev"}
	if !ev.IsTestMode {
		t.Error("isTestMode should be true when Version == \"dev\"")
	}
}

func TestIsTestMode_ReleaseVersion(t *testing.T) {
	rctx, _, ok := canSend(fullCtx("v1.7.1"))
	if !ok {
		t.Fatal("canSend returned false unexpectedly")
	}
	ev := event{IsTestMode: rctx.Version == "dev"}
	if ev.IsTestMode {
		t.Error("isTestMode should be false when Version is a real release")
	}
}

// Verify public functions do not panic when prerequisites are absent.
func TestPublicFunctions_NoopWhenAbsent(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	var err error

	RecordInitialized(ctx)
	RecordCompletion(ctx, "search-users", start, &err)
	RecordError(ctx, "id", "msg", ErrorCategoryThrownException)
}
