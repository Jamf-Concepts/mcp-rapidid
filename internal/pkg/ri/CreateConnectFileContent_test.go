// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

const (
	fileEntryBody   = `{"path":"scripts/report.js","size":12,"timestamp":1795104141101,"project":"xyz1","readable":true,"writable":true}`
	folderEntryBody = `{"path":"scripts/report.js","size":0,"timestamp":1795104141101,"project":"xyz1","readable":true,"writable":true,"fileEntries":[]}`
	nonASCIIContent = "Grüße, naïve café ✓ — line one\nline two\n"
)

// connectFileServer answers the file entry lookup with lookupStatus and
// lookupBody, and records the upload.
type connectFileServer struct {
	lookupStatus  int
	lookupBody    string
	uploadStatus  int
	uploadBody    string
	lookups       int
	lookupPath    string
	lookupProject string
	uploaded      bool
	uploadQuery   map[string]string
	uploadType    string
	uploadAccept  string
	uploadContent string
}

func (s *connectFileServer) register(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	mux.HandleFunc(baseUrlPath+"/admin/connect/files/", func(w http.ResponseWriter, r *http.Request) {
		s.lookups++
		s.lookupPath = strings.TrimPrefix(r.URL.Path, baseUrlPath+"/admin/connect/files/")
		s.lookupProject = r.URL.Query().Get("project")
		w.WriteHeader(s.lookupStatus)
		w.Write([]byte(s.lookupBody))
	})
	mux.HandleFunc(baseUrlPath+"/admin/connect/files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("upload method = %s, want POST", r.Method)
		}
		s.uploaded = true
		s.uploadQuery = map[string]string{"path": r.URL.Query().Get("path"), "project": r.URL.Query().Get("project")}
		s.uploadType = r.Header.Get("Content-Type")
		s.uploadAccept = r.Header.Get("Accept")
		body, _ := io.ReadAll(r.Body)
		s.uploadContent = string(body)
		w.WriteHeader(s.uploadStatus)
		w.Write([]byte(s.uploadBody))
	})
}

func TestCreateConnectFile(t *testing.T) {
	tests := []struct {
		name         string
		input        CreateConnectFileContentInput
		server       connectFileServer
		wantErr      string
		wantUpload   bool
		wantProject  string
		wantFilePath string
	}{
		{
			name:         "new file, content sent byte for byte",
			input:        CreateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: nonASCIIContent},
			server:       connectFileServer{lookupStatus: http.StatusNotFound, lookupBody: `{"success":false,"message":"Couldn't find file:scripts/report.js","httpStatus":404}`, uploadStatus: http.StatusOK, uploadBody: fileEntryBody},
			wantUpload:   true,
			wantProject:  "xyz1",
			wantFilePath: "scripts/report.js",
		},
		{
			name:         "<Main> project, leading slash dropped",
			input:        CreateConnectFileContentInput{Project: "<Main>", Path: " /scripts/report.js ", Content: "x"},
			server:       connectFileServer{lookupStatus: http.StatusNotFound, uploadStatus: http.StatusOK, uploadBody: fileEntryBody},
			wantUpload:   true,
			wantProject:  "",
			wantFilePath: "scripts/report.js",
		},
		{
			name:    "a file already exists",
			input:   CreateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: "x"},
			server:  connectFileServer{lookupStatus: http.StatusOK, lookupBody: fileEntryBody},
			wantErr: `a file already exists at "scripts/report.js"`,
		},
		{
			name:    "a folder already exists",
			input:   CreateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: "x"},
			server:  connectFileServer{lookupStatus: http.StatusOK, lookupBody: folderEntryBody},
			wantErr: `a folder already exists at "scripts/report.js"`,
		},
		{
			name:    "the lookup fails",
			input:   CreateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: "x"},
			server:  connectFileServer{lookupStatus: http.StatusInternalServerError, lookupBody: `{}`},
			wantErr: "non-success status 500",
		},
		{
			name:       "the upload fails",
			input:      CreateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: "x"},
			server:     connectFileServer{lookupStatus: http.StatusNotFound, uploadStatus: http.StatusForbidden, uploadBody: `{}`},
			wantErr:    "non-success status 403",
			wantUpload: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			server := tt.server
			server.register(t, mux)

			_, output, err := CreateConnectFileContent(context.Background(), newReq(), tt.input)

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
			if server.lookups != 1 {
				t.Fatalf("lookups = %d, want 1", server.lookups)
			}
			if tt.wantErr != "" {
				return
			}
			if server.lookupPath != tt.wantFilePath || server.lookupProject != tt.wantProject {
				t.Fatalf("looked up %q in project %q, want %q in %q", server.lookupPath, server.lookupProject, tt.wantFilePath, tt.wantProject)
			}
			if server.uploadQuery["path"] != tt.wantFilePath || server.uploadQuery["project"] != tt.wantProject {
				t.Fatalf("uploaded to %v, want path %q project %q", server.uploadQuery, tt.wantFilePath, tt.wantProject)
			}
			if server.uploadType != "application/octet-stream" || server.uploadAccept != "application/json" {
				t.Fatalf("upload headers Content-Type %q Accept %q", server.uploadType, server.uploadAccept)
			}
			if server.uploadContent != tt.input.Content {
				t.Fatalf("uploaded content %q, want %q", server.uploadContent, tt.input.Content)
			}
			if output.Path != "scripts/report.js" || output.Size != 12 {
				t.Fatalf("unexpected output %+v", output)
			}
		})
	}
}

func TestCreateConnectFileNoSecretLeak(t *testing.T) {
	mux := setup(t)
	server := connectFileServer{lookupStatus: http.StatusNotFound, uploadStatus: http.StatusOK, uploadBody: fileEntryBody}
	server.register(t, mux)

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		CreateConnectFileContent(context.Background(), newReq(), CreateConnectFileContentInput{Project: "xyz1", Path: "scripts/report.js", Content: "x"})
	})
}
