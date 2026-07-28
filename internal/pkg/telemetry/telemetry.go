// Copyright 2026, Jamf Software LLC

package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/rimcpctx"
)

// eventPrefix is the namespace prefix for all RapidIdMcp telemetry keys and event types.
const eventPrefix = "RapidIdMcp."

// ErrorCategory is the TelemetryDeck error category enum.
type ErrorCategory string

const (
	ErrorCategoryThrownException ErrorCategory = "thrown-exception"
	ErrorCategoryUserInput       ErrorCategory = "user-input"
	ErrorCategoryAppState        ErrorCategory = "app-state"
)

// sessionKey is the unexported context key for session-scoped telemetry fields.
type sessionKey struct{}

type sessionData struct {
	sessionID     string
	clientName    string
	clientVersion string
}

// WithSession returns a new context carrying the session-scoped telemetry fields.
func WithSession(ctx context.Context, sessionID, clientName, clientVersion string) context.Context {
	return context.WithValue(ctx, sessionKey{}, sessionData{
		sessionID:     sessionID,
		clientName:    clientName,
		clientVersion: clientVersion,
	})
}

func sessionFromContext(ctx context.Context) (sessionData, bool) {
	s, ok := ctx.Value(sessionKey{}).(sessionData)
	return s, ok
}

// event is the TelemetryDeck JSON envelope.
type event struct {
	AppID      string         `json:"appID"`
	ClientUser string         `json:"clientUser"`
	SessionID  string         `json:"sessionID"`
	Type       string         `json:"type"`
	IsTestMode bool           `json:"isTestMode"`
	Payload    map[string]any `json:"payload,omitempty"`
}

// canSend returns the rimcpctx and sessionData needed to fire an event, or false
// if any required field is absent. LicenseeID, TelemetryAppID, and TelemetryNamespace
// must all be non-empty; sessionID may be empty (StdioTransport never provides one).
func canSend(ctx context.Context) (rimcpctx.Context, sessionData, bool) {
	rctx, ok := rimcpctx.FromContext(ctx)
	if !ok {
		return rimcpctx.Context{}, sessionData{}, false
	}
	if rctx.LicenseeID == "" || rctx.TelemetryAppID == "" || rctx.TelemetryNamespace == "" {
		return rimcpctx.Context{}, sessionData{}, false
	}
	sess, ok := sessionFromContext(ctx)
	if !ok {
		return rimcpctx.Context{}, sessionData{}, false
	}
	return rctx, sess, true
}

func basePayload(rctx rimcpctx.Context, clientName, clientVersion string) map[string]any {
	platform := rctx.OS
	if rctx.OS == "darwin" {
		platform = "mac"
	}
	return map[string]any{
		"TelemetryDeck.AppInfo.buildNumber":    rctx.BuildNumber,
		"TelemetryDeck.AppInfo.version":        rctx.Version,
		"TelemetryDeck.Device.operatingSystem": rctx.OS,
		"TelemetryDeck.Device.architecture":    rctx.Arch,
		"TelemetryDeck.Device.platform":        platform,
		"TelemetryDeck.Device.modelName":       platform,
		"TelemetryDeck.SDK.name":               "Jamf-Concepts/mcp-rapidid",
		"TelemetryDeck.SDK.nameAndVersion":     fmt.Sprintf("Jamf-Concepts/mcp-rapidid-%s", rctx.Version),
		"TelemetryDeck.SDK.version":            rctx.Version,
		eventPrefix + "Client.name":               clientName,
		eventPrefix + "Client.version":            clientVersion,
	}
}

func sendAsync(ctx context.Context, rctx rimcpctx.Context, sess sessionData, eventType string, extra map[string]any) {
	payload := basePayload(rctx, sess.clientName, sess.clientVersion)
	for k, v := range extra {
		payload[k] = v
	}

	ev := event{
		AppID:      rctx.TelemetryAppID,
		ClientUser: rctx.LicenseeID,
		SessionID:  sess.sessionID,
		Type:       eventType,
		IsTestMode: rctx.Version == "dev",
		Payload:    payload,
	}

	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		body, err := json.Marshal([]event{ev})
		if err != nil {
			slog.Default().Debug("telemetry marshal failed", "type", eventType, "error", err)
			return
		}

		url := fmt.Sprintf("https://nom.telemetrydeck.com/v2/namespace/%s", rctx.TelemetryNamespace)
		req, err := http.NewRequestWithContext(sendCtx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			slog.Default().Debug("telemetry request creation failed", "type", eventType, "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Default().Debug("telemetry send failed", "type", eventType, "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			slog.Default().Debug("telemetry send non-2xx", "type", eventType, "status", resp.StatusCode)
		}
	}()
}

// RecordInitialized fires RapidIdMcp.Server.initialized.
// No-op if telemetry prerequisites are absent.
func RecordInitialized(ctx context.Context) {
	rctx, sess, ok := canSend(ctx)
	if !ok {
		return
	}
	sendAsync(ctx, rctx, sess, eventPrefix + "Server.initialized", nil)
}

// RecordCompletion fires RapidIdMcp.Tool.called with duration.
// Always fires regardless of error — does NOT fire an error event.
// No-op if telemetry prerequisites are absent.
func RecordCompletion(ctx context.Context, toolName string, start time.Time, _ *error) {
	rctx, sess, ok := canSend(ctx)
	if !ok {
		return
	}
	duration := time.Since(start).Seconds()
	sendAsync(ctx, rctx, sess, eventPrefix + "Tool.called", map[string]any{
		"TelemetryDeck.Signal.durationInSeconds": fmt.Sprintf("%.3f", duration),
		eventPrefix + "Tool.name":                   toolName,
	})
}

// RecordError fires TelemetryDeck.Error.occurred.
// No-op if telemetry prerequisites are absent.
func RecordError(ctx context.Context, id, message string, category ErrorCategory) {
	rctx, sess, ok := canSend(ctx)
	if !ok {
		return
	}
	sendAsync(ctx, rctx, sess, "TelemetryDeck.Error.occurred", map[string]any{
		"TelemetryDeck.Error.id":       id,
		"TelemetryDeck.Error.message":  message,
		"TelemetryDeck.Error.category": string(category),
	})
}
