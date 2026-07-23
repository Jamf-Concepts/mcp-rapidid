// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetEntitlementForUser(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output EntitlementForUserOutput)
	}{
		{
			name: "success empty entitlements",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"resources":[],"resourceAssociations":[]}`))
			},
			assertOutput: func(t *testing.T, output EntitlementForUserOutput) {
				if len(output.Resources) != 0 {
					t.Fatalf("expected empty resources, got %d", len(output.Resources))
				}
				if len(output.ResourceAssociations) != 0 {
					t.Fatalf("expected empty resourceAssociations, got %d", len(output.ResourceAssociations))
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"resources":[`))
			},
			wantErr: true,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`not json`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/workflow/users/user-123/associations", tt.handler)
			}

			_, output, err := GetEntitlementForUser(context.Background(), newReq(), EntitlementForUserInput{Id: "user-123"})

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

func TestGetEntitlementForUserNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/workflow/users/user-123/associations", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resources":[],"resourceAssociations":[]}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetEntitlementForUser(context.Background(), newReq(), EntitlementForUserInput{Id: "user-123"})
	})
}
