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
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const createConnectFileContentToolName = "create-connect-file-content"

type CreateConnectFileContentInput struct {
	Project string `json:"project" jsonschema:"The Connect project the file goes in. Use <Main> for the Main project"`
	Path    string `json:"path" jsonschema:"The file's path in the project, including its name, for example scripts/report.js. Missing folders are created"`
	Content string `json:"content" jsonschema:"The file's text content, written exactly as given. In a .properties file, a value written as encrypt(value) is encrypted by the server"`
}

// CreateConnectFileContent writes a new file in the Connect files module. It
// refuses a path where a file or folder already exists, because the upload
// endpoint would silently replace it; see ConnectFiles.go.
func CreateConnectFileContent(ctx context.Context, req *mcp.CallToolRequest, input CreateConnectFileContentInput) (*mcp.CallToolResult, rapididentity.FileEntry, error) {
	project, err := connectFileProject(input.Project)
	if err != nil {
		return nil, rapididentity.FileEntry{}, err
	}
	path, err := cleanConnectFilePath(input.Path)
	if err != nil {
		return nil, rapididentity.FileEntry{}, err
	}

	ctx, client, sh, setupErr := ToolSetup(ctx, req, createConnectFileContentToolName)
	if setupErr != nil {
		return nil, rapididentity.FileEntry{}, setupErr
	}
	start := time.Now()
	defer telemetry.RecordCompletion(ctx, createConnectFileContentToolName, start, &err)

	sh.Logger().Info(createConnectFileContentToolName+" tool called", "project", input.Project, "path", path, "bytes", len(input.Content))

	defer func(c *rapididentity.Client) {
		if err := c.Close(); err != nil {
			LogRIError(ctx, sh, "unable to close rapididentity client", err)
		}
	}(client)

	existing, err := lookupConnectFile(ctx, client, sh, createConnectFileContentToolName, project, path)
	if err != nil {
		return nil, rapididentity.FileEntry{}, err
	}
	if existing != nil {
		kind := "file"
		if existing.FileEntries != nil {
			kind = "folder"
		}
		err = fmt.Errorf("a %s already exists at %q; use update-connect-file-content to replace a file's content", kind, path)
		LogRIError(ctx, sh, "unable to create Connect file", err)
		return nil, rapididentity.FileEntry{}, err
	}

	requestPath := "admin/connect/files?path=" + url.QueryEscape(path) + "&project=" + url.QueryEscape(project)
	headers := http.Header{}
	headers.Set("Content-Type", "application/octet-stream")
	headers.Set("Accept", "application/json")

	sh.Logger().Info("Uploading Connect file", "path", path)
	uploadRes, err := client.DoCustomRequestWithHeaders(ctx, "POST", requestPath, headers, strings.NewReader(input.Content))
	if err != nil {
		LogRIError(ctx, sh, "unable to create Connect file", err)
		return nil, rapididentity.FileEntry{}, err
	}

	defer func(res *http.Response) {
		if err := res.Body.Close(); err != nil {
			sh.Logger().Warn("issue closing response body for admin/connect/files endpoint response", "error", err)
		}
	}(uploadRes)

	if err = checkResponseStatus(uploadRes); err != nil {
		LogRIError(ctx, sh, "unable to create Connect file", err)
		return nil, rapididentity.FileEntry{}, err
	}

	uploadBody, err := io.ReadAll(uploadRes.Body)
	if err != nil {
		sh.Logger().Error("unable to read response body for admin/connect/files response", "error", err, "status", uploadRes.StatusCode)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectFileContentToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.FileEntry{}, err
	}

	sh.Logger().Debug("POST admin/connect/files response body", "body", string(uploadBody))

	var entry rapididentity.FileEntry

	err = json.Unmarshal(uploadBody, &entry)
	if err != nil {
		sh.Logger().Error("unable to unmarshal json for POST admin/connect/files response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", createConnectFileContentToolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return nil, rapididentity.FileEntry{}, err
	}

	sh.Logger().Info("Created Connect file successfully", "path", entry.Path, "size", entry.Size)

	return nil, entry, nil
}
