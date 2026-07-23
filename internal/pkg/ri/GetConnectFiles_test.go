// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func TestGetConnectFiles(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output rapididentity.GetConnectFilesOutput)
	}{
		{
			name: "success empty files",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"fileEntries":[]}`))
			},
			assertOutput: func(t *testing.T, output rapididentity.GetConnectFilesOutput) {
				if len(output.FileEntries) != 0 {
					t.Fatalf("expected empty fileEntries, got %d", len(output.FileEntries))
				}
			},
		},
		{
			name: "malformed json body on 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"fileEntries":[`))
			},
			wantErr: true,
		},
		{
			name: "null response body returns empty files",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`null`))
			},
			wantErr: false,
			assertOutput: func(t *testing.T, output rapididentity.GetConnectFilesOutput) {
				if len(output.FileEntries) != 0 {
					t.Fatalf("expected empty fileEntries, got %d", len(output.FileEntries))
				}
			},
		},
		{
			name: "path not found",
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
				mux.HandleFunc(baseUrlPath+"/admin/connect/files/scripts", tt.handler)
			}

			_, output, err := GetConnectFiles(context.Background(), newReq(), rapididentity.GetConnectFilesInput{
				Path:    "scripts",
				Project: "<Main>",
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

func TestGetConnectFilesNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/files/scripts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"fileEntries":[]}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetConnectFiles(context.Background(), newReq(), rapididentity.GetConnectFilesInput{
			Path:    "scripts",
			Project: "<Main>",
		})
	})
}
