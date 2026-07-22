// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func TestSaveConnectAction(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output rapididentity.SaveConnectActionOutput)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":"myaction","name":"myaction","version":2}`))
			},
			assertOutput: func(t *testing.T, output rapididentity.SaveConnectActionOutput) {
				if output.Action.Id != "myaction" {
					t.Fatalf("expected action id myaction, got %q", output.Action.Id)
				}
				if output.Action.Version != 2 {
					t.Fatalf("expected version 2, got %d", output.Action.Version)
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"action":`))
			},
			wantErr: true,
		},
		{
			name: "version conflict",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"message":"Version conflict"}`))
			},
			wantErr:     true,
			errContains: "Version conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/admin/connect/actions", tt.handler)
			}

			_, output, err := SaveConnectAction(context.Background(), newReq(), rapididentity.SaveConnectActionInput{
				Action: rapididentity.ActionDef{Id: "myaction", Name: "myaction"},
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

func TestSaveConnectActionNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/actions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"myaction","name":"myaction","version":2}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		SaveConnectAction(context.Background(), newReq(), rapididentity.SaveConnectActionInput{
			Action: rapididentity.ActionDef{Id: "myaction", Name: "myaction"},
		})
	})
}
