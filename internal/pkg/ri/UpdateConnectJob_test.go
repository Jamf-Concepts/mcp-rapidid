// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// currentJobBody is what GET admin/connect/jobs/staff-id returns, shaped
// like real tenant data: every field rapididentity.ConnectJob models, and an
// action with only id, name, project and args.
const currentJobBody = `{
	"name": "StaffSync", "id": "staff-id", "version": 3, "project": "sync",
	"description": "nightly", "cronSpec": "0 0 2 * * ?", "timeZone": "America/Chicago",
	"disabled": true, "skipOverlap": true, "timeoutSeconds": 600, "traceEnabled": false,
	"runExternal": false, "logRetentionDays": 30, "emailRecipients": "ops@example.com", "attachLog": true,
	"action": {"id": "step-id", "name": "StaffSync", "project": "sync",
		"args": [{"name": "logOnly", "value": "false"}, {"name": "logLevel", "value": "\"normal\""}, {"name": "fullSync"}]}
}`

// currentJobBodyUnknownField has a field rapididentity.ConnectJob does not model.
const currentJobBodyUnknownField = `{"name": "StaffSync", "id": "staff-id", "version": 3, "project": "sync",
	"description": "", "cronSpec": "", "timeZone": "UTC", "disabled": true, "skipOverlap": false, "timeoutSeconds": 0,
	"traceEnabled": false, "runExternal": false, "logRetentionDays": 0, "emailRecipients": "", "attachLog": false,
	"retryPolicy": "never",
	"action": {"id": "step-id", "name": "StaffSync", "project": "sync", "args": []}}`

func TestUpdateConnectJob(t *testing.T) {
	tests := []struct {
		name        string
		input       UpdateConnectJobInput
		getBody     string
		postStatus  int
		postBody    string
		wantErr     bool
		errContains string
		wantList    bool
		wantGet     bool
		wantPost    bool
		// wantChanged lists the fields that must differ from the current
		// job, with their expected JSON; every other field, at every depth,
		// must be sent back as it came.
		wantChanged map[string]string
	}{
		{
			name:        "description only, the rest sent back",
			input:       UpdateConnectJobInput{Name: "StaffSync", Description: strPtr("hourly")},
			getBody:     currentJobBody,
			postStatus:  http.StatusOK,
			postBody:    `{"name":"StaffSync","id":"staff-id","version":4}`,
			wantList:    true,
			wantGet:     true,
			wantPost:    true,
			wantChanged: map[string]string{"description": `"hourly"`},
		},
		{
			name:       "schedule and enable, name matched ignoring case",
			input:      UpdateConnectJobInput{Name: " staffsync ", CronSpec: strPtr("0 0 * * * ?"), TimeZone: strPtr("UTC"), Disabled: boolPtr(false)},
			getBody:    currentJobBody,
			postStatus: http.StatusOK,
			postBody:   `{"name":"StaffSync","id":"staff-id","version":4}`,
			wantList:   true,
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
			wantList:   true,
			wantGet:    true,
			wantPost:   true,
			wantChanged: map[string]string{
				"action": `{"id":"step-id","name":"StaffSync","project":"sync","disabled":false,"outputVar":"","args":[{"name":"logOnly"},{"name":"logLevel","value":"\"debug\""},{"name":"fullSync"}]}`,
			},
		},
		{
			name:       "another action set, shared argument values carry over",
			input:      UpdateConnectJobInput{Name: "StaffSync", ActionSet: strPtr("OtherAction")},
			getBody:    currentJobBody,
			postStatus: http.StatusOK,
			postBody:   `{"name":"StaffSync","id":"staff-id","version":4}`,
			wantList:   true,
			wantGet:    true,
			wantPost:   true,
			wantChanged: map[string]string{
				"action": `{"id":"step-id","name":"OtherAction","project":"sync","disabled":false,"outputVar":"","args":[{"name":"logOnly","value":"false"},{"name":"target"}]}`,
			},
		},
		{
			name:        "a field this tool does not know stops the update",
			input:       UpdateConnectJobInput{Name: "StaffSync", Description: strPtr("x")},
			getBody:     currentJobBodyUnknownField,
			wantErr:     true,
			errContains: "could not be read safely",
			wantList:    true,
			wantGet:     true,
		},
		{
			name:        "unknown argument",
			input:       UpdateConnectJobInput{Name: "StaffSync", Args: []ConnectJobArgInput{{Name: "dryRun", Value: "true"}}},
			getBody:     currentJobBody,
			wantErr:     true,
			errContains: `has no argument named "dryRun"`,
			wantList:    true,
			wantGet:     true,
		},
		{
			name:        "new action set missing",
			input:       UpdateConnectJobInput{Name: "StaffSync", ActionSet: strPtr("NoSuchSet")},
			getBody:     currentJobBody,
			wantErr:     true,
			errContains: `project "sync" has no action set named "NoSuchSet"`,
			wantList:    true,
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
			input:       UpdateConnectJobInput{Name: "missing", Description: strPtr("x")},
			wantErr:     true,
			errContains: `no Connect job named "missing"`,
			wantList:    true,
		},
		{
			name:        "stale version refused by the server",
			input:       UpdateConnectJobInput{Name: "StaffSync", Description: strPtr("x")},
			getBody:     currentJobBody,
			postStatus:  http.StatusConflict,
			postBody:    `{"success":false,"message":"The data you provided is out of date. Please refresh and try again.","httpStatus":409}`,
			wantErr:     true,
			errContains: "non-success status 409",
			wantList:    true,
			wantGet:     true,
			wantPost:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			handleConnectActionSets(mux)

			listed, got, posted := false, false, false
			var rawBody []byte
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					listed = true
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
			if listed != tt.wantList {
				t.Fatalf("job list called = %v, want %v", listed, tt.wantList)
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

			// The SDK type always writes the action's disabled and outputVar,
			// which the API leaves out; both are their defaults and the server
			// accepts them, so they are the one expected addition.
			var want map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tt.getBody), &want); err != nil {
				t.Fatal(err)
			}
			var action map[string]json.RawMessage
			if err := json.Unmarshal(want["action"], &action); err != nil {
				t.Fatal(err)
			}
			action["disabled"] = json.RawMessage(`false`)
			action["outputVar"] = json.RawMessage(`""`)
			want["action"], _ = json.Marshal(action)
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
		})
	}
}

// TestConnectJobValidationBeforeSetup points RI_HOST at a closed port: if a
// tool connected before validating its input, the error would be a
// connection failure instead of the validation message.
func TestConnectJobValidationBeforeSetup(t *testing.T) {
	t.Setenv("RI_HOST", "http://127.0.0.1:1")
	t.Setenv("RI_USER", mockUsername)
	t.Setenv("RI_PASSWORD", mockPassword)

	checks := []struct {
		name        string
		call        func() error
		errContains string
	}{
		{"get, blank name", func() error {
			_, _, err := GetConnectJob(context.Background(), newReq(), GetConnectJobInput{Name: " "})
			return err
		}, "name is required"},
		{"run, blank name", func() error {
			_, _, err := RunConnectJob(context.Background(), newReq(), RunConnectJobInput{Name: " "})
			return err
		}, "name is required"},
		{"delete, blank name", func() error {
			_, _, err := DeleteConnectJob(context.Background(), newReq(), DeleteConnectJobInput{Name: " "})
			return err
		}, "name is required"},
		{"create, missing action set", func() error {
			_, _, err := CreateConnectJob(context.Background(), newReq(), CreateConnectJobInput{Name: "x", Project: "p"})
			return err
		}, "are required"},
		{"update, blank name", func() error {
			_, _, err := UpdateConnectJob(context.Background(), newReq(), UpdateConnectJobInput{Name: " ", Description: strPtr("x")})
			return err
		}, "name is required"},
		{"update, nothing to change", func() error {
			_, _, err := UpdateConnectJob(context.Background(), newReq(), UpdateConnectJobInput{Name: "x"})
			return err
		}, "nothing to update"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			err := c.call()
			if err == nil || !strings.Contains(err.Error(), c.errContains) {
				t.Fatalf("expected an error containing %q, got %v", c.errContains, err)
			}
		})
	}
}
