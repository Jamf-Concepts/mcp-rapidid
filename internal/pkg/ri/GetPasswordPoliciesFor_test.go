// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func TestGetPasswordPoliciesFor(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output rapididentity.PasswordPolicy)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":"policy-1","name":"Default Policy","enabled":true}`))
			},
			assertOutput: func(t *testing.T, output rapididentity.PasswordPolicy) {
				if output.Id != "policy-1" {
					t.Fatalf("expected id policy-1, got %q", output.Id)
				}
				if !output.Enabled {
					t.Fatal("expected enabled to be true")
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":`))
			},
			wantErr: true,
		},
		{
			name: "null response body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`null`))
			},
			wantErr: false,
		},
		{
			name: "user not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Not Found"}`))
			},
			wantErr:     true,
			errContains: "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/profiles/passwordPolicies/for", tt.handler)
			}

			_, output, err := GetPasswordPoliciesFor(context.Background(), newReq(), rapididentity.GetPasswordPoliciesForInput{
				UserIds: rapididentity.StringList{"user-123"},
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

func TestGetPasswordPoliciesForNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/profiles/passwordPolicies/for", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"policy-1","name":"Default Policy","enabled":true}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetPasswordPoliciesFor(context.Background(), newReq(), rapididentity.GetPasswordPoliciesForInput{
			UserIds: rapididentity.StringList{"user-123"},
		})
	})
}
