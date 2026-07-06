// Copyright 2026, Jamf Software LLC

package helper

import (
	"context"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Provides utilities for MCP tools
// to use.
type ToolHelper struct {
	// JSON logger for standard error.
	// Utilized for developers looking to
	// log events, and errors with the system.
	logger *slog.Logger

	// MCP logging notifications that are send to
	// the client. Utilized for providing notifications
	// on what the server is doing.
	notify *slog.Logger

	// Server session of the MCP server.
	// Provides information on what the client
	// supports along with other items.
	session *mcp.ServerSession

	// The progress token from the client.
	// Determines if the client supports
	// progress notifications
	token any
}


// Instantiates a ToolHelper
func NewToolHelper(req *mcp.CallToolRequest, loggerName string) *ToolHelper {
	var level slog.Level
	configuredLevel := os.Getenv("RI_LOG_LEVEL")
	err := level.UnmarshalText([]byte(configuredLevel))
	if err != nil {
		level = slog.LevelError
	}
	
	
	return &ToolHelper{
		logger: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})),
		notify: slog.New(mcp.NewLoggingHandler(req.Session, &mcp.LoggingHandlerOptions{
			LoggerName: loggerName,
		})),
		session: req.Session,
		token: req.Params.GetProgressToken(),
	}
}

// Provides logger.
func (th *ToolHelper) Logger() *slog.Logger {
	return th.logger
}

// Provides notifier
func (th *ToolHelper) Notify() *slog.Logger {
	return th.notify
}

// Tracks progress that is indeterminate, such as a network call.
func (th *ToolHelper) ProgressStep(ctx context.Context, progress float64, total float64, message string) {
	if th.token != nil {
		err := th.session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
			Progress: progress,
			Total: total,
			Message: message,
			ProgressToken: th.token,
		})
		if err != nil {
			th.logger.Warn("progress notification failed", "error", err)
		}
	}
}