// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetMyDelegations(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output GetMyDelegationsOutput)
	}{
		{
			name: "success empty delegations",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
			},
			assertOutput: func(t *testing.T, output GetMyDelegationsOutput) {
				if len(output.Delegations) != 0 {
					t.Fatalf("expected empty delegations, got %d", len(output.Delegations))
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{`))
			},
			wantErr: true,
		},
		{
			name: "null response body returns empty delegations",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`null`))
			},
			wantErr: false,
			assertOutput: func(t *testing.T, output GetMyDelegationsOutput) {
				if len(output.Delegations) != 0 {
					t.Fatalf("expected empty delegations, got %d", len(output.Delegations))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/profiles/delegations/my", tt.handler)
			}

			_, output, err := GetMyDelegations(context.Background(), newReq(), GetMyDelegationsInput{})

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

func TestGetMyDelegationsNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/profiles/delegations/my", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetMyDelegations(context.Background(), newReq(), GetMyDelegationsInput{})
	})
}
