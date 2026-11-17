// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// currentProjectBody is what GET admin/connect/projects/zzz1 returns. It
// carries fields rapididentity.ConnectProject does not model (extraTop and
// futureField) to prove the update sends them back untouched.
const currentProjectBody = `{
	"name": "zzz1",
	"id": "old-id",
	"description": "old description",
	"adminGroupDN": "cn=admins,o=example",
	"restPoints": {
		"disabled": false,
		"authSpec": {"anonymous": false, "basic": true},
		"restPoints": [{"id": "rp1", "path": "/hello", "actionSet": "hello", "futureField": "keep"}]
	},
	"changeCount": 12,
	"extraTop": 7
}`

func TestUpdateConnectProject(t *testing.T) {
	tests := []struct {
		name        string
		input       UpdateConnectProjectInput
		getStatus   int
		postStatus  int
		postBody    string
		wantErr     bool
		errContains string
		wantGet     bool
		wantPost    bool
	}{
		{
			name:       "success, name matched ignoring case",
			input:      UpdateConnectProjectInput{Name: " ZZZ1 ", Description: strPtr("new description")},
			getStatus:  http.StatusOK,
			postStatus: http.StatusOK,
			postBody:   `{"name":"zzz1","id":"old-id","description":"new description"}`,
			wantGet:    true,
			wantPost:   true,
		},
		{
			name:       "empty description clears it",
			input:      UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("")},
			getStatus:  http.StatusOK,
			postStatus: http.StatusOK,
			postBody:   `{"name":"zzz1","id":"old-id","description":""}`,
			wantGet:    true,
			wantPost:   true,
		},
		{
			name:       "only a group changes, description kept",
			input:      UpdateConnectProjectInput{Name: "zzz1", AdminGroupDN: strPtr("cn=new-admins,o=example")},
			getStatus:  http.StatusOK,
			postStatus: http.StatusOK,
			postBody:   `{"name":"zzz1","id":"old-id","description":"old description"}`,
			wantGet:    true,
			wantPost:   true,
		},
		{
			name:       "all groups set, one cleared",
			input:      UpdateConnectProjectInput{Name: "zzz1", AdminGroupDN: strPtr(""), OperatorGroupDN: strPtr("cn=ops,o=example"), AuditorGroupDN: strPtr("cn=audit,o=example")},
			getStatus:  http.StatusOK,
			postStatus: http.StatusOK,
			postBody:   `{"name":"zzz1","id":"old-id","description":"old description"}`,
			wantGet:    true,
			wantPost:   true,
		},
		{
			name:        "nothing to update",
			input:       UpdateConnectProjectInput{Name: "zzz1"},
			wantErr:     true,
			errContains: "nothing to update",
		},
		{
			name:        "project does not exist",
			input:       UpdateConnectProjectInput{Name: "missing", Description: strPtr("x")},
			wantErr:     true,
			errContains: `no Connect project named "missing"`,
		},
		{
			name:        "blank name",
			input:       UpdateConnectProjectInput{Name: "  ", Description: strPtr("x")},
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "reading the current project fails",
			input:       UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("x")},
			getStatus:   http.StatusInternalServerError,
			wantErr:     true,
			errContains: "non-success status",
			wantGet:     true,
		},
		{
			name:        "error status with valid json body",
			input:       UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("x")},
			getStatus:   http.StatusOK,
			postStatus:  http.StatusForbidden,
			postBody:    `{}`,
			wantErr:     true,
			errContains: "non-success status",
			wantGet:     true,
			wantPost:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)

			got, posted := false, false
			var postedBody map[string]json.RawMessage
			mux.HandleFunc(baseUrlPath+"/admin/connect/projects", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"projects":[{"name":"other"},{"name":"zzz1","id":"old-id"}]}`))
					return
				}
				posted = true
				reqBody, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(reqBody, &postedBody); err != nil {
					t.Errorf("request body is not JSON: %q", reqBody)
				}
				w.WriteHeader(tt.postStatus)
				w.Write([]byte(tt.postBody))
			})
			mux.HandleFunc(baseUrlPath+"/admin/connect/projects/zzz1", func(w http.ResponseWriter, r *http.Request) {
				got = true
				w.WriteHeader(tt.getStatus)
				w.Write([]byte(currentProjectBody))
			})

			_, output, err := UpdateConnectProject(context.Background(), newReq(), tt.input)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
			}
			if got != tt.wantGet {
				t.Fatalf("GET of the current project called = %v, want %v", got, tt.wantGet)
			}
			if posted != tt.wantPost {
				t.Fatalf("POST called = %v, want %v", posted, tt.wantPost)
			}
			if tt.wantErr || !tt.wantPost {
				return
			}

			// The body must be the whole current project with only the
			// description changed: same id, same name spelling, every
			// other field (including unmodelled ones) as it was.
			var want map[string]json.RawMessage
			if err := json.Unmarshal([]byte(currentProjectBody), &want); err != nil {
				t.Fatal(err)
			}
			for field, value := range map[string]*string{
				"description":     tt.input.Description,
				"adminGroupDN":    tt.input.AdminGroupDN,
				"operatorGroupDN": tt.input.OperatorGroupDN,
				"auditorGroupDN":  tt.input.AuditorGroupDN,
			} {
				if value != nil {
					encoded, _ := json.Marshal(*value)
					want[field] = encoded
				}
			}
			if got, wantCompact := compactFields(t, postedBody), compactFields(t, want); !reflect.DeepEqual(got, wantCompact) {
				t.Fatalf("posted body\n got %v\nwant %v", got, wantCompact)
			}
			if output.Id != "old-id" {
				t.Fatalf("unexpected output %+v", output)
			}
		})
	}
}

func TestUpdateConnectProjectNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/projects", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"projects":[{"name":"zzz1"}]}`))
			return
		}
		w.Write([]byte(`{"name":"zzz1","id":"old-id"}`))
	})
	mux.HandleFunc(baseUrlPath+"/admin/connect/projects/zzz1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(currentProjectBody))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		UpdateConnectProject(context.Background(), newReq(), UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("x")})
	})
}

func strPtr(s string) *string {
	return &s
}

// compactFields returns each raw JSON field without insignificant
// whitespace, so that bodies can be compared field by field.
func compactFields(t *testing.T, fields map[string]json.RawMessage) map[string]string {
	t.Helper()
	compacted := make(map[string]string, len(fields))
	for name, raw := range fields {
		var buf bytes.Buffer
		if err := json.Compact(&buf, raw); err != nil {
			t.Fatalf("field %q is not valid JSON: %v", name, err)
		}
		compacted[name] = buf.String()
	}
	return compacted
}
