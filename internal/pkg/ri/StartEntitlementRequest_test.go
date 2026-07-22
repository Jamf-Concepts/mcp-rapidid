// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestStartEntitlementRequest(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output StartEntitlementRequestOutput)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`["request-id-1","request-id-2"]`))
			},
			assertOutput: func(t *testing.T, output StartEntitlementRequestOutput) {
				if len(output.RequestIds) != 2 {
					t.Fatalf("expected 2 request ids, got %d", len(output.RequestIds))
				}
				if output.RequestIds[0] != "request-id-1" {
					t.Fatalf("expected request-id-1, got %q", output.RequestIds[0])
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`["request-id-1"`))
			},
			wantErr: true,
		},
		{
			name: "null response body returns empty request ids",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`null`))
			},
			wantErr: false,
			assertOutput: func(t *testing.T, output StartEntitlementRequestOutput) {
				if len(output.RequestIds) != 0 {
					t.Fatalf("expected empty request ids, got %d", len(output.RequestIds))
				}
			},
		},
		{
			name: "unauthorized",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"message":"Unauthorized"}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/workflow/tasks/startTask", tt.handler)
			}

			_, output, err := StartEntitlementRequest(context.Background(), newReq(), StartEntitlementRequestInput{
				RequestInfo: []StartEntitlementRequestInfo{
					{
						Type:       "GRANT",
						UserId:     "user-123",
						ResourceId: "resource-456",
					},
				},
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

func TestStartEntitlementRequestNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/workflow/tasks/startTask", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`["request-id-1"]`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		StartEntitlementRequest(context.Background(), newReq(), StartEntitlementRequestInput{
			RequestInfo: []StartEntitlementRequestInfo{
				{
					Type:       "GRANT",
					UserId:     "user-123",
					ResourceId: "resource-456",
				},
			},
		})
	})
}
