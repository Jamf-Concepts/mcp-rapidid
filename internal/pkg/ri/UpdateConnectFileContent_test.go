// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestUpdateConnectFile(t *testing.T) {
	tests := []struct {
		name       string
		input      UpdateConnectFileContentInput
		server     connectFileServer
		wantErr    string
		wantUpload bool
	}{
		{
			name:       "existing file replaced byte for byte",
			input:      UpdateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: nonASCIIContent},
			server:     connectFileServer{lookupStatus: http.StatusOK, lookupBody: fileEntryBody, uploadStatus: http.StatusOK, uploadBody: fileEntryBody},
			wantUpload: true,
		},
		{
			name:       "empty content empties the file",
			input:      UpdateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: ""},
			server:     connectFileServer{lookupStatus: http.StatusOK, lookupBody: fileEntryBody, uploadStatus: http.StatusOK, uploadBody: fileEntryBody},
			wantUpload: true,
		},
		{
			name:    "no file at the path",
			input:   UpdateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: "x"},
			server:  connectFileServer{lookupStatus: http.StatusNotFound},
			wantErr: `no file exists at "scripts/report.js"`,
		},
		{
			name:    "a folder at the path",
			input:   UpdateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: "x"},
			server:  connectFileServer{lookupStatus: http.StatusOK, lookupBody: folderEntryBody},
			wantErr: `"scripts/report.js" is a folder`,
		},
		{
			name:       "the upload fails",
			input:      UpdateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: "x"},
			server:     connectFileServer{lookupStatus: http.StatusOK, lookupBody: fileEntryBody, uploadStatus: http.StatusForbidden, uploadBody: `{}`},
			wantErr:    "non-success status 403",
			wantUpload: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			server := tt.server
			server.register(t, mux)

			_, output, err := UpdateConnectFileContent(context.Background(), newReq(), tt.input)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected an error containing %q, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if server.uploaded != tt.wantUpload {
				t.Fatalf("upload called = %v, want %v", server.uploaded, tt.wantUpload)
			}
			if tt.wantErr != "" {
				return
			}
			if server.uploadQuery["path"] != "scripts/report.js" || server.uploadQuery["project"] != "xyz1" {
				t.Fatalf("uploaded to %v", server.uploadQuery)
			}
			if server.uploadType != "application/octet-stream" {
				t.Fatalf("upload Content-Type %q, want application/octet-stream", server.uploadType)
			}
			if server.uploadContent != tt.input.Content {
				t.Fatalf("uploaded content %q, want %q", server.uploadContent, tt.input.Content)
			}
			if output.Path != "scripts/report.js" {
				t.Fatalf("unexpected output %+v", output)
			}
		})
	}
}

// TestConnectFileContentValidationBeforeSetup points RI_HOST at a closed port: if a
// file tool connected before validating its input, the error would be a
// connection failure instead of the validation message.
func TestConnectFileContentValidationBeforeSetup(t *testing.T) {
	t.Setenv("RI_HOST", "http://127.0.0.1:1")
	t.Setenv("RI_USER", mockUsername)
	t.Setenv("RI_PASSWORD", mockPassword)

	bad := []struct {
		project string
		path    string
		want    string
	}{
		{"", "a.txt", "a project is required"},
		{"xyz1", "  ", "a file path is required"},
		{"xyz1", "scripts/", "is a folder"},
		{"xyz1", "a/../b.txt", `".." segment`},
		{"xyz1", "./a.txt", `".." segment`},
		{"xyz1", "a//b.txt", `".." segment`},
	}
	for _, b := range bad {
		_, _, createErr := CreateConnectFileContent(context.Background(), newReq(), CreateConnectFileContentInput{Project: b.project, Path: b.path})
		_, _, updateErr := UpdateConnectFileContent(context.Background(), newReq(), UpdateConnectFileContentInput{Project: b.project, Path: b.path})
		for tool, err := range map[string]error{"create": createErr, "update": updateErr} {
			if err == nil || !strings.Contains(err.Error(), b.want) {
				t.Errorf("%s with project %q path %q: expected an error containing %q, got %v", tool, b.project, b.path, b.want, err)
			}
		}
	}
}
