// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"
)

// currentProjectBody is what GET admin/connect/projects/zzz1 returns, shaped
// like real tenant data. Its REST points carry the fields that
// rapididentity.ConnectProject v1.7.0 does not model (consumes, a REST
// point's own authSpec, argMap sourceKey), and fields the API sometimes
// leaves out are left out here too, to prove a round trip sends them back
// exactly as they came.
const currentProjectBody = `{
	"name": "zzz1",
	"id": "old-id",
	"description": "old description",
	"adminGroupDN": "cn=admins,o=example",
	"operatorGroupDN": "",
	"auditorGroupDN": "",
	"restPoints": {
		"disabled": false,
		"authSpec": {"anonymous": false, "oauth1": false, "basic": true, "basicWithOAuthKeys": false},
		"restPoints": [
			{"id": "rp1", "description": "provisioning", "method": "POST", "disabled": false, "path": "/provisioning",
			 "consumes": "application/json", "produces": "application/json", "actionSet": "provision",
			 "authSpec": {"anonymous": true, "oauth1": false, "basic": false, "basicWithOAuthKeys": false},
			 "argMap": [{"sourceKey": "userId", "sourceType": "QUERY", "destKey": "user", "destType": "STRING"},
			            {"sourceType": "CONTENT", "destType": "OBJECT", "destKey": "reportInfo"},
			            {"destType": "BOOLEAN", "destKey": "successful"}]},
			{"id": "rp2", "description": "status", "method": "GET", "disabled": true, "path": "/status",
			 "produces": "text/plain", "actionSet": "status"}
		]
	},
	"changeCount": 12,
	"modifiedMs": 1794608853699,
	"modifiedBy": "403ab949-684e-4a06-862c-231e2a165b89",
	"modifiedByName": "BChrist"
}`

// projectWithoutRestPoints has no restPoints key at all, as two of thirty
// tenant projects do.
const projectWithoutRestPoints = `{"name":"zzz1","id":"old-id","description":"","adminGroupDN":"","operatorGroupDN":"","auditorGroupDN":"","changeCount":1,"modifiedMs":1,"modifiedBy":"x","modifiedByName":"y"}`

// projectWithUnknownField has a REST point field ConnectProject does not have.
const projectWithUnknownField = `{"name":"zzz1","id":"old-id","description":"","adminGroupDN":"","operatorGroupDN":"","auditorGroupDN":"",
	"restPoints":{"disabled":false,"authSpec":{"anonymous":false,"oauth1":false,"basic":false,"basicWithOAuthKeys":false},
	"restPoints":[{"id":"rp1","description":"","method":"GET","disabled":false,"path":"/x","produces":"","actionSet":"x","rateLimit":5}]},
	"changeCount":1,"modifiedMs":1,"modifiedBy":"x","modifiedByName":"y"}`

func TestUpdateConnectProject(t *testing.T) {
	tests := []struct {
		name        string
		input       UpdateConnectProjectInput
		getBody     string
		getStatus   int
		postStatus  int
		postBody    string
		wantErr     bool
		errContains string
		wantList    bool
		wantGet     bool
		wantPost    bool
		// wantChanged lists the fields that must differ from the current
		// project, with their expected JSON; every other field, at every
		// depth, must be sent back as it came.
		wantChanged map[string]string
	}{
		{
			name:        "success, name matched ignoring case, REST points kept",
			input:       UpdateConnectProjectInput{Name: " ZZZ1 ", Description: strPtr("new description")},
			getBody:     currentProjectBody,
			getStatus:   http.StatusOK,
			postStatus:  http.StatusOK,
			postBody:    `{"name":"zzz1","id":"old-id","description":"new description"}`,
			wantList:    true,
			wantGet:     true,
			wantPost:    true,
			wantChanged: map[string]string{"description": `"new description"`},
		},
		{
			name:        "empty description clears it",
			input:       UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("")},
			getBody:     currentProjectBody,
			getStatus:   http.StatusOK,
			postStatus:  http.StatusOK,
			postBody:    `{"name":"zzz1","id":"old-id","description":""}`,
			wantList:    true,
			wantGet:     true,
			wantPost:    true,
			wantChanged: map[string]string{"description": `""`},
		},
		{
			name:        "all groups set, one cleared, description kept",
			input:       UpdateConnectProjectInput{Name: "zzz1", AdminGroupDN: strPtr(""), OperatorGroupDN: strPtr("cn=ops,o=example"), AuditorGroupDN: strPtr("cn=audit,o=example")},
			getBody:     currentProjectBody,
			getStatus:   http.StatusOK,
			postStatus:  http.StatusOK,
			postBody:    `{"name":"zzz1","id":"old-id"}`,
			wantList:    true,
			wantGet:     true,
			wantPost:    true,
			wantChanged: map[string]string{"adminGroupDN": `""`, "operatorGroupDN": `"cn=ops,o=example"`, "auditorGroupDN": `"cn=audit,o=example"`},
		},
		{
			name:        "project without REST points stays without them",
			input:       UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("x")},
			getBody:     projectWithoutRestPoints,
			getStatus:   http.StatusOK,
			postStatus:  http.StatusOK,
			postBody:    `{"name":"zzz1","id":"old-id","description":"x"}`,
			wantList:    true,
			wantGet:     true,
			wantPost:    true,
			wantChanged: map[string]string{"description": `"x"`},
		},
		{
			name:        "a field this tool does not know stops the update",
			input:       UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("x")},
			getBody:     projectWithUnknownField,
			getStatus:   http.StatusOK,
			wantErr:     true,
			errContains: "could not be read safely",
			wantList:    true,
			wantGet:     true,
		},
		{
			name:        "nothing to update, refused before connecting",
			input:       UpdateConnectProjectInput{Name: "zzz1"},
			wantErr:     true,
			errContains: "nothing to update",
		},
		{
			name:        "blank name, refused before connecting",
			input:       UpdateConnectProjectInput{Name: "  ", Description: strPtr("x")},
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "project does not exist",
			input:       UpdateConnectProjectInput{Name: "missing", Description: strPtr("x")},
			wantErr:     true,
			errContains: `no Connect project named "missing"`,
			wantList:    true,
		},
		{
			name:        "reading the current project fails",
			input:       UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("x")},
			getBody:     currentProjectBody,
			getStatus:   http.StatusInternalServerError,
			wantErr:     true,
			errContains: "non-success status",
			wantList:    true,
			wantGet:     true,
		},
		{
			name:        "error status with valid json body",
			input:       UpdateConnectProjectInput{Name: "zzz1", Description: strPtr("x")},
			getBody:     currentProjectBody,
			getStatus:   http.StatusOK,
			postStatus:  http.StatusForbidden,
			postBody:    `{}`,
			wantErr:     true,
			errContains: "non-success status",
			wantList:    true,
			wantGet:     true,
			wantPost:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)

			listed, got, posted := false, false, false
			var rawBody []byte
			mux.HandleFunc(baseUrlPath+"/admin/connect/projects", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					listed = true
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"projects":[{"name":"other"},{"name":"zzz1","id":"old-id"}]}`))
					return
				}
				posted = true
				rawBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(tt.postStatus)
				w.Write([]byte(tt.postBody))
			})
			mux.HandleFunc(baseUrlPath+"/admin/connect/projects/zzz1", func(w http.ResponseWriter, r *http.Request) {
				got = true
				w.WriteHeader(tt.getStatus)
				w.Write([]byte(tt.getBody))
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
			if listed != tt.wantList {
				t.Fatalf("project list called = %v, want %v", listed, tt.wantList)
			}
			if got != tt.wantGet {
				t.Fatalf("GET of the current project called = %v, want %v", got, tt.wantGet)
			}
			if posted != tt.wantPost {
				t.Fatalf("POST called = %v, want %v", posted, tt.wantPost)
			}
			if tt.wantErr {
				return
			}

			var want map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tt.getBody), &want); err != nil {
				t.Fatal(err)
			}
			for field, value := range tt.wantChanged {
				want[field] = json.RawMessage(value)
			}
			wantJSON, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			if gotCanon, wantCanon := canonicalJSON(t, rawBody), canonicalJSON(t, wantJSON); gotCanon != wantCanon {
				t.Fatalf("posted body\n got %s\nwant %s", gotCanon, wantCanon)
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

// canonicalJSON rewrites a JSON document with every object's keys sorted and
// no insignificant whitespace, so that two documents can be compared without
// depending on key order. Objects and arrays are walked as raw JSON, so the
// values themselves are never decoded into loosely typed Go values.
func canonicalJSON(t *testing.T, raw []byte) string {
	t.Helper()
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		t.Fatal("empty JSON value")
	}
	switch raw[0] {
	case '{':
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatalf("not a JSON object: %s (%v)", raw, err)
		}
		names := make([]string, 0, len(fields))
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		var buf strings.Builder
		buf.WriteByte('{')
		for i, name := range names {
			if i > 0 {
				buf.WriteByte(',')
			}
			key, _ := json.Marshal(name)
			buf.Write(key)
			buf.WriteByte(':')
			buf.WriteString(canonicalJSON(t, fields[name]))
		}
		buf.WriteByte('}')
		return buf.String()
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("not a JSON array: %s (%v)", raw, err)
		}
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = canonicalJSON(t, item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		var buf bytes.Buffer
		if err := json.Compact(&buf, raw); err != nil {
			t.Fatalf("not valid JSON: %s (%v)", raw, err)
		}
		return buf.String()
	}
}
