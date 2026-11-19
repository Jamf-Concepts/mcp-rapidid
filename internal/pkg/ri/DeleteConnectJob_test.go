// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestDeleteConnectJob(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		deleteStatus int
		deleteBody   string
		wantErr      bool
		errContains  string
		wantDelete   bool
	}{
		{
			name:         "success, name matched ignoring case",
			input:        " STAFFSYNC ",
			deleteStatus: http.StatusOK,
			deleteBody:   `{"success":true,"message":"","httpStatus":200}`,
			wantDelete:   true,
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
			name:         "success false on 200",
			input:        "StaffSync",
			deleteStatus: http.StatusOK,
			deleteBody:   `{"success":false,"message":"job is in use","httpStatus":200}`,
			wantErr:      true,
			errContains:  "job is in use",
			wantDelete:   true,
		},
		{
			name:         "error status with valid json body",
			input:        "StaffSync",
			deleteStatus: http.StatusForbidden,
			deleteBody:   `{"success":true}`,
			wantErr:      true,
			errContains:  "non-success status",
			wantDelete:   true,
		},
		{
			name:         "malformed json body on 200",
			input:        "StaffSync",
			deleteStatus: http.StatusOK,
			deleteBody:   `{"success":`,
			wantErr:      true,
			wantDelete:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)

			deleted := false
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(connectJobsBody))
			})
			// Registered by id: the job endpoints do not find a job by its
			// bare name, so a request by name would not reach this handler.
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs/staff-id", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("method = %s, want DELETE", r.Method)
					return
				}
				deleted = true
				w.WriteHeader(tt.deleteStatus)
				w.Write([]byte(tt.deleteBody))
			})

			_, output, err := DeleteConnectJob(context.Background(), newReq(), DeleteConnectJobInput{Name: tt.input})

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
			}
			if deleted != tt.wantDelete {
				t.Fatalf("DELETE called = %v, want %v", deleted, tt.wantDelete)
			}
			if !tt.wantErr && !output.DeleteOperationStatus.Success {
				t.Fatal("expected success to be true")
			}
		})
	}
}

func TestDeleteConnectJobNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(connectJobsBody))
	})
	mux.HandleFunc(baseUrlPath+"/admin/connect/jobs/staff-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"message":"","httpStatus":200}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		DeleteConnectJob(context.Background(), newReq(), DeleteConnectJobInput{Name: "StaffSync"})
	})
}
