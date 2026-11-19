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

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

// staffSyncActionSet is what GET admin/connect/actions/sync.StaffSync returns.
const staffSyncActionSet = `{"name":"StaffSync","project":"sync","id":"as-id","argDefs":[{"name":"logOnly","type":"boolean"},{"name":"logLevel"},{"name":"fullSync"}]}`

// otherActionSet is what GET admin/connect/actions/sync.OtherAction returns.
const otherActionSet = `{"name":"OtherAction","project":"sync","id":"other-as-id","argDefs":[{"name":"logOnly"},{"name":"target"}]}`

func handleConnectActionSets(mux *http.ServeMux) {
	for path, body := range map[string]string{"sync.StaffSync": staffSyncActionSet, "sync.OtherAction": otherActionSet} {
		mux.HandleFunc(baseUrlPath+"/admin/connect/actions/"+path, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(body))
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestCreateConnectJob(t *testing.T) {
	tests := []struct {
		name        string
		input       CreateConnectJobInput
		postStatus  int
		postBody    string
		wantErr     bool
		errContains string
		wantPost    bool
		wantBody    createConnectJobBody
	}{
		{
			name:       "success, disabled by default, every argument listed",
			input:      CreateConnectJobInput{Name: " NewJob ", Project: "sync", ActionSet: "StaffSync", Args: []ConnectJobArgInput{{Name: "logOnly", Value: "true"}}, CronSpec: "0 0 2 * * ?", TimeZone: "America/Chicago", Description: "nightly"},
			postStatus: http.StatusOK,
			postBody:   `{"name":"NewJob","id":"new-id","version":1,"disabled":true}`,
			wantPost:   true,
			wantBody: createConnectJobBody{Name: "NewJob", Project: "sync", Description: "nightly", CronSpec: "0 0 2 * * ?", TimeZone: "America/Chicago", Disabled: true,
				Action: createConnectJobAction{Name: "StaffSync", Project: "sync", Args: rapididentity.ArgDefList{{Name: "logOnly", Value: "true"}, {Name: "logLevel"}, {Name: "fullSync"}}}},
		},
		{
			name:       "explicitly enabled",
			input:      CreateConnectJobInput{Name: "NewJob", Project: "sync", ActionSet: "StaffSync", Disabled: boolPtr(false)},
			postStatus: http.StatusOK,
			postBody:   `{"name":"NewJob","id":"new-id","version":1}`,
			wantPost:   true,
			wantBody: createConnectJobBody{Name: "NewJob", Project: "sync", Disabled: false,
				Action: createConnectJobAction{Name: "StaffSync", Project: "sync", Args: rapididentity.ArgDefList{{Name: "logOnly"}, {Name: "logLevel"}, {Name: "fullSync"}}}},
		},
		{
			name:        "name taken, ignoring case",
			input:       CreateConnectJobInput{Name: "staffsync", Project: "sync", ActionSet: "StaffSync"},
			wantErr:     true,
			errContains: `a Connect job named "StaffSync" already exists`,
		},
		{
			name:        "action set missing",
			input:       CreateConnectJobInput{Name: "NewJob", Project: "sync", ActionSet: "NoSuchSet"},
			wantErr:     true,
			errContains: `project "sync" has no action set named "NoSuchSet"`,
		},
		{
			name:        "unknown argument",
			input:       CreateConnectJobInput{Name: "NewJob", Project: "sync", ActionSet: "StaffSync", Args: []ConnectJobArgInput{{Name: "dryRun", Value: "true"}}},
			wantErr:     true,
			errContains: `has no argument named "dryRun"`,
		},
		{
			name:        "missing action set name",
			input:       CreateConnectJobInput{Name: "NewJob", Project: "sync"},
			wantErr:     true,
			errContains: "are required",
		},
		{
			name:        "server error status",
			input:       CreateConnectJobInput{Name: "NewJob", Project: "sync", ActionSet: "StaffSync"},
			postStatus:  http.StatusInternalServerError,
			postBody:    `{"success":false,"message":"java.lang.IllegalArgumentException: Duplicate job name","httpStatus":500}`,
			wantErr:     true,
			errContains: "non-success status 500",
			wantPost:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			handleConnectActionSets(mux)

			posted := false
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

			_, _, err := CreateConnectJob(context.Background(), newReq(), tt.input)

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
			if tt.wantErr {
				return
			}

			var fields map[string]json.RawMessage
			if err := json.Unmarshal(rawBody, &fields); err != nil {
				t.Fatalf("request body is not JSON: %q", rawBody)
			}
			for _, key := range []string{"id", "version"} {
				if _, ok := fields[key]; ok {
					t.Fatalf("a new job must not send %q: %s", key, rawBody)
				}
			}
			var action map[string]json.RawMessage
			if err := json.Unmarshal(fields["action"], &action); err != nil {
				t.Fatalf("action is not a JSON object: %s", fields["action"])
			}
			if _, ok := action["id"]; ok {
				t.Fatalf("a new job's action must not send an id: %s", fields["action"])
			}
			var got createConnectJobBody
			if err := json.Unmarshal(rawBody, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.wantBody) {
				t.Fatalf("posted body\n got %+v\nwant %+v", got, tt.wantBody)
			}
		})
	}
}
