// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetUserInfoInDelegation(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output UserInfoInDelegationOutput)
	}{
		{
			name: "success empty profiles",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"adminLimitEnforced":false,"profiles":[]}`))
			},
			assertOutput: func(t *testing.T, output UserInfoInDelegationOutput) {
				if len(output.Profiles) != 0 {
					t.Fatalf("expected empty profiles, got %d", len(output.Profiles))
				}
				if output.AdminLimitEnforced {
					t.Fatal("expected adminLimitEnforced to be false")
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"profiles":[`))
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
				mux.HandleFunc(baseUrlPath+"/profiles/delegations/my/deleg-123/profiles/searchByFilter", tt.handler)
			}

			_, output, err := GetUserInfoInDelegation(context.Background(), newReq(), UserInfoInDelegationInput{
				DelegationId: "deleg-123",
				Filter:       "(uid=testuser)",
			})

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

func TestGetUserInfoInDelegationNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/profiles/delegations/my/deleg-123/profiles/searchByFilter", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"adminLimitEnforced":false,"profiles":[]}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetUserInfoInDelegation(context.Background(), newReq(), UserInfoInDelegationInput{
			DelegationId: "deleg-123",
			Filter:       "(uid=testuser)",
		})
	})
}
