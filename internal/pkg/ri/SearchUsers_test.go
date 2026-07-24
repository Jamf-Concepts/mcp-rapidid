// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestSearchRapidIdentityUsers(t *testing.T) {
	tests := []struct {
		name              string
		delegationHandler http.HandlerFunc
		usersHandler      http.HandlerFunc
		wantErr           bool
		errContains       string
		assertOutput      func(t *testing.T, output UserOutput)
	}{
		{
			name: "success empty users",
			delegationHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"id":"deleg-1","name":"My Delegation","type":"MY"}]`))
			},
			usersHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
			},
			assertOutput: func(t *testing.T, output UserOutput) {
				if len(output.Users) != 0 {
					t.Fatalf("expected empty users, got %d", len(output.Users))
				}
			},
		},
		{
			name: "delegation call fails",
			delegationHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"message":"Internal Server Error"}`))
			},
			wantErr: true,
		},
		{
			name: "malformed delegation json",
			delegationHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{`))
			},
			wantErr: true,
		},
		{
			name: "users call fails",
			delegationHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"id":"deleg-1","name":"My Delegation","type":"MY"}]`))
			},
			usersHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"message":"Internal Server Error"}`))
			},
			wantErr: true,
		},
		{
			name: "malformed users json",
			delegationHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"id":"deleg-1","name":"My Delegation","type":"MY"}]`))
			},
			usersHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.delegationHandler != nil {
				mux.HandleFunc(baseUrlPath+"/profiles/delegations/my", tt.delegationHandler)
			}
			if tt.usersHandler != nil {
				mux.HandleFunc(baseUrlPath+"/users", tt.usersHandler)
			}

			_, output, err := SearchRapidIdentityUsers(context.Background(), newReq(), UserInput{Criteria: "testuser"})

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

func TestSearchRapidIdentityUsersNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/profiles/delegations/my", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":"deleg-1","name":"My Delegation","type":"MY"}]`))
	})
	mux.HandleFunc(baseUrlPath+"/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		SearchRapidIdentityUsers(context.Background(), newReq(), UserInput{Criteria: "testuser"})
	})
}

func TestSearchRapidIdentityUsersServiceIdentity(t *testing.T) {
	tests := []struct {
		name         string
		usersHandler http.HandlerFunc
		wantErr      bool
		assertOutput func(t *testing.T, output UserOutput)
	}{
		{
			name: "success returns users from reporting endpoint",
			usersHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"users":[{"id":"idauto-1","username":"jdoe","firstName":"Jane","lastName":"Doe"}],"adminLimitEnforced":false}`))
			},
			assertOutput: func(t *testing.T, output UserOutput) {
				if len(output.Users) != 1 {
					t.Fatalf("expected 1 user, got %d", len(output.Users))
				}
				if output.Users[0].Username != "jdoe" {
					t.Fatalf("expected username jdoe, got %q", output.Users[0].Username)
				}
			},
		},
		{
			name: "success empty users",
			usersHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"users":[],"adminLimitEnforced":false}`))
			},
			assertOutput: func(t *testing.T, output UserOutput) {
				if len(output.Users) != 0 {
					t.Fatalf("expected empty users, got %d", len(output.Users))
				}
			},
		},
		{
			name: "malformed reporting json",
			usersHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			t.Setenv("RI_SERVICE_IDENTITY_SECRET_KEY", "svc-secret")
			if tt.usersHandler != nil {
				mux.HandleFunc(baseUrlPath+"/reporting/users", tt.usersHandler)
			}

			_, output, err := SearchRapidIdentityUsers(context.Background(), newReq(), UserInput{Criteria: "testuser"})

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.assertOutput != nil {
				tt.assertOutput(t, output)
			}
		})
	}
}

func TestSearchRapidIdentityUsersServiceIdentityNoSecretLeak(t *testing.T) {
	serviceKey := "svc-secret-do-not-log"
	mux := setup(t)
	t.Setenv("RI_SERVICE_IDENTITY_SECRET_KEY", serviceKey)
	mux.HandleFunc(baseUrlPath+"/reporting/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users":[],"adminLimitEnforced":false}`))
	})

	assertNoSecretLeak(t, []string{serviceKey}, func() {
		SearchRapidIdentityUsers(context.Background(), newReq(), UserInput{Criteria: "testuser"})
	})
}
