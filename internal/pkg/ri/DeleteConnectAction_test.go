// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func TestDeleteConnectAction(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output rapididentity.DeleteConnectActionByIdOutput)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success":true,"message":"","httpStatus":200}`))
			},
			assertOutput: func(t *testing.T, output rapididentity.DeleteConnectActionByIdOutput) {
				if !output.DeleteOperationStatus.Success {
					t.Fatal("expected success to be true")
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"deleteOperationStatus":`))
			},
			wantErr: true,
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
				mux.HandleFunc(baseUrlPath+"/admin/connect/actions/myaction", tt.handler)
			}

			_, output, err := DeleteConnectAction(context.Background(), newReq(), rapididentity.DeleteConnectActionByIdInput{Id: "myaction"})

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

func TestDeleteConnectActionNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/actions/myaction", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"message":"","httpStatus":200}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		DeleteConnectAction(context.Background(), newReq(), rapididentity.DeleteConnectActionByIdInput{Id: "myaction"})
	})
}
