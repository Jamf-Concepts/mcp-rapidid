// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func TestGetConnectFileContent(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output GetConnectFileContentOutput)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`console.log("hello")`))
			},
			assertOutput: func(t *testing.T, output GetConnectFileContentOutput) {
				if output.Content != `console.log("hello")` {
					t.Fatalf("expected file content, got %q", output.Content)
				}
			},
		},
		{
			name: "empty file content",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(``))
			},
			assertOutput: func(t *testing.T, output GetConnectFileContentOutput) {
				if output.Content != "" {
					t.Fatalf("expected empty content, got %q", output.Content)
				}
			},
		},
		{
			name: "file not found",
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
				mux.HandleFunc(baseUrlPath+"/admin/connect/fileContent/myfile.js", tt.handler)
			}

			_, output, err := GetConnectFileContent(context.Background(), newReq(), rapididentity.GetConnectFileContentInput{
				Path:    "myfile.js",
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

func TestGetConnectFileContentNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/admin/connect/fileContent/myfile.js", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`console.log("hello")`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetConnectFileContent(context.Background(), newReq(), rapididentity.GetConnectFileContentInput{
			Path:    "myfile.js",
			Project: "<Main>",
		})
	})
}
