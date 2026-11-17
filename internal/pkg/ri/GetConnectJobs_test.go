// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

// connectJobsBody is what GET admin/connect/jobs returns in these tests.
const connectJobsBody = `{"jobs":[
	{"name":"OtherJob","id":"other-id","project":"other","version":1},
	{"name":"StaffSync","id":"staff-id","project":"sync","version":3,"cronSpec":"0 0 2 * * ?","timeZone":"America/Chicago","disabled":false,
	 "action":{"id":"a1","name":"StaffSync","project":"sync","args":[{"name":"logOnly"},{"name":"logLevel","value":"\"normal\""}]}}
]}`

func TestGetConnectJobs(t *testing.T) {
	tests := []struct {
		name         string
		input        rapididentity.GetConnectJobsInput
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output rapididentity.GetConnectJobsOutput)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "" {
					t.Errorf("expected no query, got %q", r.URL.RawQuery)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(connectJobsBody))
			},
			assertOutput: func(t *testing.T, output rapididentity.GetConnectJobsOutput) {
				if len(output.Jobs) != 2 || output.Jobs[1].Name != "StaffSync" || output.Jobs[1].CronSpec != "0 0 2 * * ?" {
					t.Fatalf("unexpected jobs %+v", output.Jobs)
				}
			},
		},
		{
			name:  "filtered by project",
			input: rapididentity.GetConnectJobsInput{Project: "sync"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("project"); got != "sync" {
					t.Errorf("project query = %q, want sync", got)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jobs":[]}`))
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jobs": [`))
			},
			wantErr: true,
		},
		{
			name: "null response body returns empty jobs",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`null`))
			},
			assertOutput: func(t *testing.T, output rapididentity.GetConnectJobsOutput) {
				if len(output.Jobs) != 0 {
					t.Fatalf("expected empty jobs, got %d", len(output.Jobs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", tt.handler)

			_, output, err := GetConnectJobs(context.Background(), newReq(), tt.input)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
			}
			if tt.assertOutput != nil {
				tt.assertOutput(t, output)
			}
		})
	}
}

func TestGetConnectJobsNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(connectJobsBody))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetConnectJobs(context.Background(), newReq(), rapididentity.GetConnectJobsInput{})
	})
}
