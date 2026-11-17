// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestDeleteConnectProject(t *testing.T) {
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
			input:        " ZZZ1 ",
			deleteStatus: http.StatusOK,
			deleteBody:   `{"success":true,"message":"","httpStatus":200}`,
			wantDelete:   true,
		},
		{
			name:        "project does not exist",
			input:       "missing",
			wantErr:     true,
			errContains: `no Connect project named "missing"`,
		},
		{
			name:        "blank name",
			input:       "  ",
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:         "success false on 200",
			input:        "zzz1",
			deleteStatus: http.StatusOK,
			deleteBody:   `{"success":false,"message":"project is in use","httpStatus":200}`,
			wantErr:      true,
			errContains:  "project is in use",
			wantDelete:   true,
		},
		{
			name:         "error status with valid json body",
			input:        "zzz1",
			deleteStatus: http.StatusForbidden,
			deleteBody:   `{"success":true}`,
			wantErr:      true,
			errContains:  "non-success status",
			wantDelete:   true,
		},
		{
			name:         "malformed json body on 200",
			input:        "zzz1",
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
			mux.HandleFunc(baseUrlPath+"/admin/connect/projects", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"projects":[{"name":"other"},{"name":"zzz1","id":"old-id"}]}`))
			})
			// Registered by exact name: a request for "ZZZ1" or "old-id"
			// would not reach this handler.
			mux.HandleFunc(baseUrlPath+"/admin/connect/projects/zzz1", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("method = %s, want DELETE", r.Method)
				}
				deleted = true
				w.WriteHeader(tt.deleteStatus)
				w.Write([]byte(tt.deleteBody))
			})

			_, output, err := DeleteConnectProject(context.Background(), newReq(), DeleteConnectProjectInput{Name: tt.input})

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

func TestDeleteConnectProjectNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/projects", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"projects":[{"name":"zzz1"}]}`))
	})
	mux.HandleFunc(baseUrlPath+"/admin/connect/projects/zzz1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"message":"","httpStatus":200}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		DeleteConnectProject(context.Background(), newReq(), DeleteConnectProjectInput{Name: "zzz1"})
	})
}
