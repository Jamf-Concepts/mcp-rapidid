// Copyright 2026, Jamf Software LLC

package helper

import (
	"context"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Provides utilities for the MCP Server
type ServerHelper struct {
	// JSON logger for standard error.
	// Utilized for developers looking to
	// log events, and errors with the system.
	logger *slog.Logger

	// The log level of the logger.
	logLevel *slog.LevelVar

	// Server session of the MCP server.
	// Provides information on what the client
	// supports along with other items.
	session *mcp.ServerSession

	// The progress token from the client.
	// Determines if the client supports
	// progress notifications
	token any

	// The name of the tool this helper was created for.
	// Used to build structured telemetry error IDs.
	toolName string

	// The name of the prompt this helper was created for.
	// Used to build structured telemetry error IDs.
	promptName string
}

// Instantiates a ToolHelper
func NewToolServerHelper(req *mcp.CallToolRequest, toolName string) *ServerHelper {
	levelVar := new(slog.LevelVar)
	configuredLevel := os.Getenv("RI_LOG_LEVEL")
	err := levelVar.UnmarshalText([]byte(configuredLevel))
	if err != nil {
		levelVar.Set(slog.LevelError)
	}

	return &ServerHelper{
		logger:   slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: levelVar.Level()})),
		logLevel: levelVar,
		session:  req.Session,
		token:    req.Params.GetProgressToken(),
		toolName: toolName,
	}
}

// Instantiates a PromptHelper
func NewPromptServerHelper(req *mcp.GetPromptRequest, promptName string) *ServerHelper {
	levelVar := new(slog.LevelVar)
	configuredLevel := os.Getenv("RI_LOG_LEVEL")
	err := levelVar.UnmarshalText([]byte(configuredLevel))
	if err != nil {
		levelVar.Set(slog.LevelError)
	}

	return &ServerHelper{
		logger:     slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: levelVar.Level()})),
		logLevel:   levelVar,
		session:    req.Session,
		token:      req.Params.GetProgressToken(),
		promptName: promptName,
	}
}

// Provides logger.
func (sh *ServerHelper) Logger() *slog.Logger {
	return sh.logger
}

// Tracks progress of known process length
func (sh *ServerHelper) ProgressStep(ctx context.Context, progress float64, total float64, message string) {
	if sh.token != nil {
		err := sh.session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
			Progress:      progress,
			Total:         total,
			Message:       message,
			ProgressToken: sh.token,
		})
		if err != nil {
			sh.logger.Warn("progress notification failed", "error", err)
		}
	}
}

// Retrieve the log level of logger
func (sh *ServerHelper) LogLevel() slog.Level {
	return sh.logLevel.Level()
}

// ToolName returns the name of the tool this helper was created for.
func (sh *ServerHelper) ToolName() string {
	return sh.toolName
}

// PromptName returns the name of the tool this helper was created for.
func (sh *ServerHelper) PromptName() string {
	return sh.promptName
}
