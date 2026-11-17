// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetConnectJob(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		getStatus   int
		getBody     string
		wantErr     bool
		errContains string
		wantGet     bool
	}{
		{
			name:      "success, name matched ignoring case",
			input:     " staffsync ",
			getStatus: http.StatusOK,
			getBody:   `{"name":"StaffSync","id":"staff-id","project":"sync","version":3,"cronSpec":"0 0 2 * * ?","action":{"name":"StaffSync","args":[{"name":"logOnly"}]}}`,
			wantGet:   true,
		},
		{
			name:        "job does not exist",
			input:       "missing",
			wantErr:     true,
			errContains: `no Connect job named "missing"`,
		},
		{
			name:        "blank name",
			input:       "  ",
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "error status with valid json body",
			input:       "StaffSync",
			getStatus:   http.StatusForbidden,
			getBody:     `{}`,
			wantErr:     true,
			errContains: "non-success status",
			wantGet:     true,
		},
		{
			name:      "malformed json body on 200",
			input:     "StaffSync",
			getStatus: http.StatusOK,
			getBody:   `{"name":`,
			wantErr:   true,
			wantGet:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)

			got := false
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(connectJobsBody))
			})
			// Registered by id: the job endpoints do not find a job by its
			// bare name, so a request by name would not reach this handler.
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs/staff-id", func(w http.ResponseWriter, r *http.Request) {
				got = true
				w.WriteHeader(tt.getStatus)
				w.Write([]byte(tt.getBody))
			})

			_, output, err := GetConnectJob(context.Background(), newReq(), GetConnectJobInput{Name: tt.input})

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
				t.Fatalf("GET of the job called = %v, want %v", got, tt.wantGet)
			}
			if !tt.wantErr && (output.Id != "staff-id" || output.Version != 3 || output.CronSpec != "0 0 2 * * ?" || output.Action.Name != "StaffSync") {
				t.Fatalf("unexpected output %+v", output)
			}
		})
	}
}

func TestGetConnectJobNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(connectJobsBody))
	})
	mux.HandleFunc(baseUrlPath+"/admin/connect/jobs/staff-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"StaffSync","id":"staff-id"}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetConnectJob(context.Background(), newReq(), GetConnectJobInput{Name: "StaffSync"})
	})
}
