// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestRunConnectJob(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		runStatus   int
		runBody     string
		wantErr     bool
		errContains string
		wantRun     bool
	}{
		{
			name:      "success, name matched ignoring case",
			input:     " STAFFSYNC ",
			runStatus: http.StatusOK,
			runBody:   `{"success":true,"message":"","httpStatus":200}`,
			wantRun:   true,
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
			name:        "success false on 200",
			input:       "StaffSync",
			runStatus:   http.StatusOK,
			runBody:     `{"success":false,"message":"job is disabled","httpStatus":200}`,
			wantErr:     true,
			errContains: "job is disabled",
			wantRun:     true,
		},
		{
			name:        "error status with valid json body",
			input:       "StaffSync",
			runStatus:   http.StatusForbidden,
			runBody:     `{"success":true}`,
			wantErr:     true,
			errContains: "non-success status",
			wantRun:     true,
		},
		{
			name:      "malformed json body on 200",
			input:     "StaffSync",
			runStatus: http.StatusOK,
			runBody:   `{"success":`,
			wantErr:   true,
			wantRun:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)

			ran := false
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(connectJobsBody))
			})
			// Registered by id: the job endpoints do not find a job by its
			// bare name, so a request by name would not reach this handler.
			mux.HandleFunc(baseUrlPath+"/admin/connect/runJob/staff-id", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", r.Method)
				}
				ran = true
				w.WriteHeader(tt.runStatus)
				w.Write([]byte(tt.runBody))
			})

			_, output, err := RunConnectJob(context.Background(), newReq(), RunConnectJobInput{Name: tt.input})

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
			}
			if ran != tt.wantRun {
				t.Fatalf("runJob called = %v, want %v", ran, tt.wantRun)
			}
			if !tt.wantErr && !output.RunOperationStatus.Success {
				t.Fatal("expected success to be true")
			}
		})
	}
}

func TestRunConnectJobNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(connectJobsBody))
	})
	mux.HandleFunc(baseUrlPath+"/admin/connect/runJob/staff-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"message":"","httpStatus":200}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		RunConnectJob(context.Background(), newReq(), RunConnectJobInput{Name: "StaffSync"})
	})
}
