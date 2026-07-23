// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	baseUrlPath          = "/api/rest"
	mockUsername         = "rapidid"
	mockPassword         = "NOTAREALPASSWORD"
	mockFailAuthUsername = "failauth"
)

func newReq() *mcp.CallToolRequest {
	return &mcp.CallToolRequest{
		Session: &mcp.ServerSession{},
		Params:  &mcp.CallToolParamsRaw{},
	}
}

func setup(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	baseUrl, _ := url.Parse(server.URL)
	t.Setenv("RI_HOST", baseUrl.String())
	t.Setenv("RI_USER", mockUsername)
	t.Setenv("RI_PASSWORD", mockPassword)

	mux.HandleFunc(baseUrlPath+"/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
		}

		if r.Method == http.MethodPost {
			var user rapididentity.RapidIdentityUser
			reqBody, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body.Close()

			err = json.Unmarshal(reqBody, &user)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if user.Username == mockFailAuthUsername {
				w.WriteHeader(401)
				w.Write(json.RawMessage(`{"message": "Authentication Failed"}`))
				return
			}

			session := rapididentity.Session{
				Session: rapididentity.SessionInfo{
					Token: "abcd",
				},
			}
			res, _ := json.Marshal(session)
			w.WriteHeader(http.StatusOK)
			w.Write(res)
		}
	})

	t.Cleanup(server.Close)

	return mux
}

func assertNoSecretLeak(t *testing.T, secrets []string, toolFn func()) {
	t.Helper()
	t.Setenv("RI_LOG_LEVEL", "DEBUG")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w

	toolFn()

	w.Close()
	os.Stderr = orig

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	captured := buf.String()
	for _, secret := range secrets {
		if strings.Contains(captured, secret) {
			t.Fatalf("secret leaked in log output: %q", secret)
		}
	}
}

func TestToolSetup(t *testing.T) {
	tests := []struct {
		name        string
		envOverride func(t *testing.T)
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
	}{
		{
			name: "bad user credentials",
			envOverride: func(t *testing.T) {
				t.Setenv("RI_USER", mockFailAuthUsername)
			},
			wantErr:     true,
			errContains: "Authentication Failed",
		},
		{
			name: "bad service identity",
			envOverride: func(t *testing.T) {
				t.Setenv("RI_SERVICE_IDENTITY_SECRET_KEY", "badsecret")
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"message":"Authentication Failed"}`))
			},
			wantErr:     true,
			errContains: "Authentication Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.envOverride != nil {
				tt.envOverride(t)
			}
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/admin/connect/projects", tt.handler)
			}

			_, _, err := GetConnectProjects(context.Background(), newReq(), GetConnectProjectsInput{})

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}
