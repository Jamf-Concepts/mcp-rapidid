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

// currentJobBody is what GET admin/connect/jobs/staff-id returns. futureTop
// is a field rapididentity.ConnectJob does not model, to prove an update
// sends it back untouched.
const currentJobBody = `{
	"name": "StaffSync", "id": "staff-id", "version": 3, "project": "sync",
	"description": "nightly", "cronSpec": "0 0 2 * * ?", "timeZone": "America/Chicago",
	"disabled": true, "skipOverlap": true, "timeoutSeconds": 600, "traceEnabled": false,
	"runExternal": false, "logRetentionDays": 30, "emailRecipients": "ops@example.com", "attachLog": true,
	"futureTop": 7,
	"action": {"id": "step-id", "name": "StaffSync", "project": "sync",
		"args": [{"name": "logOnly", "value": "false"}, {"name": "logLevel", "value": "\"normal\""}, {"name": "fullSync"}]}
}`

// currentJobBodyOddAction has an action field this tool does not model.
const currentJobBodyOddAction = `{"name": "StaffSync", "id": "staff-id", "version": 3, "project": "sync",
	"action": {"id": "step-id", "name": "StaffSync", "project": "sync", "outputVar": "x", "args": []}}`

func TestUpdateConnectJob(t *testing.T) {
	tests := []struct {
		name        string
		input       UpdateConnectJobInput
		getBody     string
		postStatus  int
		postBody    string
		wantErr     bool
		errContains string
		wantGet     bool
		wantPost    bool
		// wantChanged lists the fields that must differ from the current
		// job, with their expected JSON; every other field must be equal.
		wantChanged map[string]string
	}{
		{
			name:        "description only, action sent back untouched",
			input:       UpdateConnectJobInput{Name: "StaffSync", Description: jobStrPtr("hourly")},
			getBody:     currentJobBody,
			postStatus:  http.StatusOK,
			postBody:    `{"name":"StaffSync","id":"staff-id","version":4}`,
			wantGet:     true,
			wantPost:    true,
			wantChanged: map[string]string{"description": `"hourly"`},
		},
		{
			name:       "schedule and enable",
			input:      UpdateConnectJobInput{Name: "staffsync", CronSpec: jobStrPtr("0 0 * * * ?"), TimeZone: jobStrPtr("UTC"), Disabled: boolPtr(false)},
			getBody:    currentJobBody,
			postStatus: http.StatusOK,
			postBody:   `{"name":"StaffSync","id":"staff-id","version":4}`,
			wantGet:    true,
			wantPost:   true,
			wantChanged: map[string]string{
				"cronSpec": `"0 0 * * * ?"`, "timeZone": `"UTC"`, "disabled": `false`,
			},
		},
		{
			name:       "arguments merged by name, empty value clears",
			input:      UpdateConnectJobInput{Name: "StaffSync", Args: []ConnectJobArgInput{{Name: "logLevel", Value: `"debug"`}, {Name: "logOnly", Value: ""}}},
			getBody:    currentJobBody,
			postStatus: http.StatusOK,
			postBody:   `{"name":"StaffSync","id":"staff-id","version":4}`,
			wantGet:    true,
			wantPost:   true,
			wantChanged: map[string]string{
				"action": `{"id":"step-id","name":"StaffSync","project":"sync","args":[{"name":"logOnly"},{"name":"logLevel","value":"\"debug\""},{"name":"fullSync"}]}`,
			},
		},
		{
			name:       "another action set, shared argument values carry over",
			input:      UpdateConnectJobInput{Name: "StaffSync", ActionSet: jobStrPtr("OtherAction")},
			getBody:    currentJobBody,
			postStatus: http.StatusOK,
			postBody:   `{"name":"StaffSync","id":"staff-id","version":4}`,
			wantGet:    true,
			wantPost:   true,
			wantChanged: map[string]string{
				"action": `{"id":"step-id","name":"OtherAction","project":"sync","args":[{"name":"logOnly","value":"false"},{"name":"target"}]}`,
			},
		},
		{
			name:        "description change keeps an unknown action field",
			input:       UpdateConnectJobInput{Name: "StaffSync", Description: jobStrPtr("x")},
			getBody:     currentJobBodyOddAction,
			postStatus:  http.StatusOK,
			postBody:    `{"name":"StaffSync","id":"staff-id","version":4}`,
			wantGet:     true,
			wantPost:    true,
			wantChanged: map[string]string{"description": `"x"`},
		},
		{
			name:        "action change refused for an unknown action field",
			input:       UpdateConnectJobInput{Name: "StaffSync", Args: []ConnectJobArgInput{{Name: "logOnly", Value: "true"}}},
			getBody:     currentJobBodyOddAction,
			wantErr:     true,
			errContains: "shape this tool does not know",
			wantGet:     true,
		},
		{
			name:        "unknown argument",
			input:       UpdateConnectJobInput{Name: "StaffSync", Args: []ConnectJobArgInput{{Name: "dryRun", Value: "true"}}},
			getBody:     currentJobBody,
			wantErr:     true,
			errContains: `has no argument named "dryRun"`,
			wantGet:     true,
		},
		{
			name:        "new action set missing",
			input:       UpdateConnectJobInput{Name: "StaffSync", ActionSet: jobStrPtr("NoSuchSet")},
			getBody:     currentJobBody,
			wantErr:     true,
			errContains: `project "sync" has no action set named "NoSuchSet"`,
			wantGet:     true,
		},
		{
			name:        "nothing to update",
			input:       UpdateConnectJobInput{Name: "StaffSync"},
			wantErr:     true,
			errContains: "nothing to update",
		},
		{
			name:        "job does not exist",
			input:       UpdateConnectJobInput{Name: "missing", Description: jobStrPtr("x")},
			wantErr:     true,
			errContains: `no Connect job named "missing"`,
		},
		{
			name:        "stale version refused by the server",
			input:       UpdateConnectJobInput{Name: "StaffSync", Description: jobStrPtr("x")},
			getBody:     currentJobBody,
			postStatus:  http.StatusConflict,
			postBody:    `{"success":false,"message":"The data you provided is out of date. Please refresh and try again.","httpStatus":409}`,
			wantErr:     true,
			errContains: "out of date",
			wantGet:     true,
			wantPost:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			handleConnectActionSets(mux)

			got, posted := false, false
			var rawBody []byte
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(connectJobsBody))
					return
				}
				posted = true
				rawBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(tt.postStatus)
				w.Write([]byte(tt.postBody))
			})
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs/staff-id", func(w http.ResponseWriter, r *http.Request) {
				got = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.getBody))
			})

			_, _, err := UpdateConnectJob(context.Background(), newReq(), tt.input)

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
				t.Fatalf("GET of the current job called = %v, want %v", got, tt.wantGet)
			}
			if posted != tt.wantPost {
				t.Fatalf("POST called = %v, want %v", posted, tt.wantPost)
			}
			if tt.wantErr {
				return
			}

			var current, sent map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tt.getBody), &current); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(rawBody, &sent); err != nil {
				t.Fatalf("request body is not JSON: %q", rawBody)
			}
			want := compactRawFields(t, current)
			for field, value := range tt.wantChanged {
				want[field] = compactRaw(t, []byte(value))
			}
			if gotFields := compactRawFields(t, sent); !reflect.DeepEqual(gotFields, want) {
				t.Fatalf("posted body\n got %v\nwant %v", gotFields, want)
			}
		})
	}
}

// compactRawFields returns each raw JSON field without insignificant
// whitespace, so that bodies can be compared field by field.
func compactRawFields(t *testing.T, fields map[string]json.RawMessage) map[string]string {
	t.Helper()
	compacted := make(map[string]string, len(fields))
	for name, raw := range fields {
		compacted[name] = compactRaw(t, raw)
	}
	return compacted
}

func compactRaw(t *testing.T, raw []byte) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		t.Fatalf("not valid JSON: %s (%v)", raw, err)
	}
	return buf.String()
}
