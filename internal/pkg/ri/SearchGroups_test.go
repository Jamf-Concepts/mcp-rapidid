// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestSearchGroups(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output SearchGroupsOutput)
	}{
		{
			name: "success empty results",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"users":[],"groups":[],"adminLimitEnforced":false}`))
			},
			assertOutput: func(t *testing.T, output SearchGroupsOutput) {
				if len(output.Groups) != 0 {
					t.Fatalf("expected empty groups, got %d", len(output.Groups))
				}
				if len(output.Users) != 0 {
					t.Fatalf("expected empty users, got %d", len(output.Users))
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"groups":[`))
			},
			wantErr: true,
		},
		{
			name: "null response body returns empty results",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`null`))
			},
			wantErr: false,
			assertOutput: func(t *testing.T, output SearchGroupsOutput) {
				if len(output.Groups) != 0 {
					t.Fatalf("expected empty groups, got %d", len(output.Groups))
				}
				if len(output.Users) != 0 {
					t.Fatalf("expected empty users, got %d", len(output.Users))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/roles/managedGroups/searchTask", tt.handler)
			}

			_, output, err := SearchGroups(context.Background(), newReq(), SearchGroupsInput{Criteria: "testgroup"})

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

func TestSearchGroupsNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/roles/managedGroups/searchTask", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users":[],"groups":[],"adminLimitEnforced":false}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		SearchGroups(context.Background(), newReq(), SearchGroupsInput{Criteria: "testgroup"})
	})
}
