// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestConnectJobLookupCase covers two jobs whose names differ only in case,
// which the server allows: the exact name must win, and a name that matches
// both only ignoring case must be refused rather than guessed.
func TestConnectJobLookupCase(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPath    string
		errContains string
	}{
		{name: "exact name, mixed case", input: "StaffSync", wantPath: "/admin/connect/jobs/staff-id"},
		{name: "exact name, upper case", input: "STAFFSYNC", wantPath: "/admin/connect/jobs/upper-id"},
		{name: "matches both ignoring case", input: "staffsync", errContains: "more than one Connect job matches"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)

			gotPath := ""
			mux.HandleFunc(baseUrlPath+"/admin/connect/jobs", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jobs":[{"name":"StaffSync","id":"staff-id"},{"name":"STAFFSYNC","id":"upper-id"}]}`))
			})
			for _, id := range []string{"staff-id", "upper-id"} {
				mux.HandleFunc(baseUrlPath+"/admin/connect/jobs/"+id, func(w http.ResponseWriter, r *http.Request) {
					gotPath = strings.TrimPrefix(r.URL.Path, baseUrlPath)
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"name":"x","id":"` + id + `"}`))
				})
			}

			_, _, err := GetConnectJob(context.Background(), newReq(), GetConnectJobInput{Name: tt.input})

			if tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected an error containing %q, got %v", tt.errContains, err)
				}
				if gotPath != "" {
					t.Fatalf("no job should be fetched, got %s", gotPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("fetched %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}
