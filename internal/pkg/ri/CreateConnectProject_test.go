// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const existingProjectsBody = `{"projects":[{"name":"Existing Project","id":"existing-id"}]}`

func TestCreateConnectProject(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		groups       CreateConnectProjectInput
		wantBody     map[string]string
		listBody     string
		postStatus   int
		postBody     string
		wantErr      bool
		errContains  string
		wantPost     bool
		assertOutput func(t *testing.T, output ConnectProject)
	}{
		{
			name:       "success",
			input:      "  New Project ",
			listBody:   existingProjectsBody,
			postStatus: http.StatusOK,
			postBody:   `{"name":"New Project","id":"new-id"}`,
			wantPost:   true,
			assertOutput: func(t *testing.T, output ConnectProject) {
				if output.Name != "New Project" || output.Id != "new-id" {
					t.Fatalf("expected New Project/new-id, got %q/%q", output.Name, output.Id)
				}
			},
		},
		{
			name:       "with groups, empty ones left out",
			input:      "New Project",
			groups:     CreateConnectProjectInput{AdminGroupDN: "cn=admins,o=example", AuditorGroupDN: "cn=audit,o=example"},
			listBody:   existingProjectsBody,
			postStatus: http.StatusOK,
			postBody:   `{"name":"New Project","id":"new-id"}`,
			wantPost:   true,
			wantBody:   map[string]string{"name": "New Project", "adminGroupDN": "cn=admins,o=example", "auditorGroupDN": "cn=audit,o=example"},
		},
		{
			name:        "blank name",
			input:       "   ",
			listBody:    existingProjectsBody,
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "name already taken, ignoring case",
			input:       "existing project",
			listBody:    existingProjectsBody,
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name:       "malformed json body on 200",
			input:      "New Project",
			listBody:   existingProjectsBody,
			postStatus: http.StatusOK,
			postBody:   `{"name":`,
			wantErr:    true,
			wantPost:   true,
		},
		{
			name:        "error status with valid json body",
			input:       "New Project",
			listBody:    existingProjectsBody,
			postStatus:  http.StatusForbidden,
			postBody:    `{}`,
			wantErr:     true,
			errContains: "non-success status",
			wantPost:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)

			var postedName string
			var postedBody map[string]string
			posted := false
			mux.HandleFunc(baseUrlPath+"/admin/connect/projects", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.listBody))
					return
				}
				posted = true
				reqBody, _ := io.ReadAll(r.Body)
				postedBody = decodeStringFields(t, reqBody)
				postedName = postedBody["name"]
				w.WriteHeader(tt.postStatus)
				w.Write([]byte(tt.postBody))
			})

			input := tt.groups
			input.Name = tt.input
			_, output, err := CreateConnectProject(context.Background(), newReq(), input)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
			}
			if posted != tt.wantPost {
				t.Fatalf("POST called = %v, want %v", posted, tt.wantPost)
			}
			if tt.wantPost && postedName != strings.TrimSpace(tt.input) {
				t.Fatalf("posted name %q, want %q", postedName, strings.TrimSpace(tt.input))
			}
			if tt.wantPost {
				wantBody := tt.wantBody
				if wantBody == nil {
					wantBody = map[string]string{"name": strings.TrimSpace(tt.input)}
				}
				if !reflect.DeepEqual(postedBody, wantBody) {
					t.Fatalf("posted body %v, want %v", postedBody, wantBody)
				}
			}
			if tt.assertOutput != nil {
				tt.assertOutput(t, output)
			}
		})
	}
}

func TestCreateConnectProjectNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/projects", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"projects":[]}`))
			return
		}
		w.Write([]byte(`{"name":"New Project","id":"new-id"}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		CreateConnectProject(context.Background(), newReq(), CreateConnectProjectInput{Name: "New Project"})
	})
}

// decodeStringFields decodes a JSON object whose values must all be strings,
// failing the test otherwise.
func decodeStringFields(t *testing.T, body []byte) map[string]string {
	t.Helper()
	var fields map[string]string
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Errorf("request body is not an object of strings: %q (%v)", body, err)
	}
	return fields
}
