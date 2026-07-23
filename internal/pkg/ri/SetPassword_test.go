// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func TestSetPassword(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output SetPasswordOutput)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"target":"user-123","success":true,"targetName":"Test User"}]`))
			},
			assertOutput: func(t *testing.T, output SetPasswordOutput) {
				if len(output.Result) != 1 {
					t.Fatalf("expected 1 result, got %d", len(output.Result))
				}
				if !output.Result[0].Success {
					t.Fatal("expected success to be true")
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
			name: "unauthorized",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"message":"Unauthorized"}`))
			},
			wantErr:     true,
			errContains: "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/profiles/actions/password", tt.handler)
			}

			_, output, err := SetPassword(context.Background(), newReq(), rapididentity.SetPasswordInput{
				Targets:     rapididentity.StringList{"user-123"},
				NewPassword: "newpassword123",
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

func TestSetPasswordNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/profiles/actions/password", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"target":"user-123","success":true,"targetName":"Test User"}]`))
	})

	const newPassword = "newpassword123"

	assertNoSecretLeak(t, []string{mockPassword, "abcd", newPassword}, func() {
		SetPassword(context.Background(), newReq(), rapididentity.SetPasswordInput{
			Targets:     rapididentity.StringList{"user-123"},
			NewPassword: newPassword,
		})
	})
}
