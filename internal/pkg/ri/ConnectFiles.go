// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

// The Connect files endpoints behave like this (observed against a live
// tenant):
//
//   - POST /admin/connect/files?project=P&path=dir/name.ext uploads the raw
//     request body as the file. The path includes the file name, and missing
//     folders are created. The content round-trips byte for byte.
//   - The upload SILENTLY REPLACES an existing file. So create-connect-file-content
//     refuses a path that already exists, and update-connect-file-content refuses
//     one that does not, rather than overwriting or creating by accident.
//   - GET /admin/connect/files/{path}?project=P returns the file's entry, or
//     404 when nothing exists at the path. A folder's entry lists its
//     fileEntries; a file's has none.
//   - In a .properties file such as Globals.properties, a value written as
//     encrypt(value) is encrypted by the server on upload and stored as
//     decrypt(...), the same as a save in the UI.
//   - Replies are XML unless the request asks for application/json.

// connectFileEntry is the entry GET /admin/connect/files/{path} returns.
// FileEntries is set only for a folder.
type connectFileEntry struct {
	rapididentity.FileEntry
	FileEntries *[]rapididentity.FileEntry `json:"fileEntries"`
}

// connectFileProject returns the project query value for a file request:
// the project name, or "" for the <Main> project. A project is required so
// that a write cannot land in <Main> because the caller left it out.
func connectFileProject(project string) (string, error) {
	project = strings.TrimSpace(project)
	switch project {
	case "":
		return "", fmt.Errorf("a project is required; use %s for the Main project", rapididentity.MainProject)
	case rapididentity.MainProject:
		return "", nil
	}
	return project, nil
}

// cleanConnectFilePath returns path without surrounding spaces or a leading
// slash, refusing an empty path, a folder path (trailing slash) and any "."
// or ".." segment.
func cleanConnectFilePath(path string) (string, error) {
	path = strings.TrimPrefix(strings.TrimSpace(path), "/")
	if path == "" {
		return "", fmt.Errorf("a file path is required")
	}
	if strings.HasSuffix(path, "/") {
		return "", fmt.Errorf("the path %q is a folder; give a file path including its name", path)
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("the path %q has an empty, \".\" or \"..\" segment", path)
		}
	}
	return path, nil
}

// escapeConnectFilePath escapes each segment of path for use in a URL path,
// keeping the slashes between them.
func escapeConnectFilePath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

// lookupConnectFile returns the entry at path in project, or nil when
// nothing exists there.
func lookupConnectFile(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, toolName string, project string, path string) (*connectFileEntry, error) {
	requestPath := "admin/connect/files/" + escapeConnectFilePath(path) + "?project=" + url.QueryEscape(project)
	headers := http.Header{}
	headers.Set("Accept", "application/json")

	sh.Logger().Info("Checking for an existing Connect file", "path", path)
	res, err := client.DoCustomRequestWithHeaders(ctx, "GET", requestPath, headers, nil)
	if err != nil {
		LogRIError(ctx, sh, "unable to check for Connect file", err)
		return nil, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for admin/connect/files endpoint response", "error", err)
		}
	}(res)

	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err = checkResponseStatus(res); err != nil {
		LogRIError(ctx, sh, "unable to check for Connect file", err)
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for admin/connect/files response", "error", err, "status", res.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", toolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, err
	}

	var entry connectFileEntry
	if err = json.Unmarshal(body, &entry); err != nil {
		sh.Logger().Error("unable to unmarshal json for admin/connect/files response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", toolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, err
	}
	return &entry, nil
}
