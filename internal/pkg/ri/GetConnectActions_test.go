// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func TestGetConnectActions(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output rapididentity.GetConnectActionsOutput)
	}{
		{
			name: "success empty actions",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"name":"all","actionDefs":[]}`))
			},
			assertOutput: func(t *testing.T, output rapididentity.GetConnectActionsOutput) {
				if output.Name != "all" {
					t.Fatalf("expected name all, got %q", output.Name)
				}
				if len(output.ActionDefs) != 0 {
					t.Fatalf("expected empty actionDefs, got %d", len(output.ActionDefs))
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"actionDefs":[`))
			},
			wantErr: true,
		},
		{
			name: "null response body returns empty actions",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`null`))
			},
			wantErr: false,
			assertOutput: func(t *testing.T, output rapididentity.GetConnectActionsOutput) {
				if len(output.ActionDefs) != 0 {
					t.Fatalf("expected empty actionDefs, got %d", len(output.ActionDefs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/admin/connect/actions", tt.handler)
			}

			_, output, err := GetConnectActions(context.Background(), newReq(), rapididentity.GetConnectActionsInput{})

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

func TestGetConnectActionsNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/actions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"all","actionDefs":[]}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetConnectActions(context.Background(), newReq(), rapididentity.GetConnectActionsInput{})
	})
}
