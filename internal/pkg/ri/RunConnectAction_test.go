// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func TestRunConnectAction(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output rapididentity.RunConnectActionOutput)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<html>Action completed</html>`))
			},
			assertOutput: func(t *testing.T, output rapididentity.RunConnectActionOutput) {
				if output.Log != "<html>Action completed</html>" {
					t.Fatalf("expected log content, got %q", output.Log)
				}
			},
		},
		{
			name: "action not authorized",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`Unauthorized`))
			},
			wantErr:     true,
			errContains: "Unauthorized",
		},
		{
			name: "action not found",
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
				mux.HandleFunc(baseUrlPath+"/admin/connect/run", tt.handler)
			}

			_, output, err := RunConnectAction(context.Background(), newReq(), rapididentity.RunConnectActionInput{
				Action: rapididentity.ConnectAction{Id: "myaction"},
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

func TestRunConnectActionNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/run", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html>Action completed</html>`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		RunConnectAction(context.Background(), newReq(), rapididentity.RunConnectActionInput{
			Action: rapididentity.ConnectAction{Id: "myaction"},
		})
	})
}
