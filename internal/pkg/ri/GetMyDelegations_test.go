// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetMyDelegations(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output GetMyDelegationsOutput)
	}{
		{
			name: "success empty delegations",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
			},
			assertOutput: func(t *testing.T, output GetMyDelegationsOutput) {
				if len(output.Delegations) != 0 {
					t.Fatalf("expected empty delegations, got %d", len(output.Delegations))
				}
			},
		},
		{
			name: "success full delegation fields",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"id":"c3b2d872-9bf4-456d-9c3b-a510c376b4ff","name":"Staff","description":"All Employee Accounts","type":"CUSTOM","preloadResults":true,"editProfileMessage":null,"layout1":"first_name","layout2":"last_name","layout3":"email","attributes":[{"galItem":{"id":"idauto_id","friendlyName":"ID","searchable":true,"multiValued":false,"allowMultiValue":false,"type":"STRING","typeParams":null},"name":"ID","editable":false,"showInList":false,"showInDetails":true,"required":false}],"actions":[{"id":"DISABLE","name":"Disable","description":"Disable the selected profiles"}]}]`))
			},
			assertOutput: func(t *testing.T, output GetMyDelegationsOutput) {
				if len(output.Delegations) != 1 {
					t.Fatalf("expected 1 delegation, got %d", len(output.Delegations))
				}
				d := output.Delegations[0]
				if d.Id != "c3b2d872-9bf4-456d-9c3b-a510c376b4ff" {
					t.Errorf("unexpected id: %s", d.Id)
				}
				if d.Name != "Staff" {
					t.Errorf("unexpected name: %s", d.Name)
				}
				if d.Description != "All Employee Accounts" {
					t.Errorf("unexpected description: %s", d.Description)
				}
				if d.Type != "CUSTOM" {
					t.Errorf("unexpected type: %s", d.Type)
				}
				if len(d.Attributes) != 1 {
					t.Fatalf("expected 1 attribute name, got %d", len(d.Attributes))
				}
				if d.Attributes[0] != "ID" {
					t.Errorf("unexpected attribute name: %s", d.Attributes[0])
				}
				if len(d.Actions) != 1 {
					t.Fatalf("expected 1 action name, got %d", len(d.Actions))
				}
				if d.Actions[0] != "Disable" {
					t.Errorf("unexpected action name: %s", d.Actions[0])
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
			name: "null response body returns empty delegations",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`null`))
			},
			wantErr: false,
			assertOutput: func(t *testing.T, output GetMyDelegationsOutput) {
				if len(output.Delegations) != 0 {
					t.Fatalf("expected empty delegations, got %d", len(output.Delegations))
				}
			},
		},
		{
			// A non-2xx response whose body is valid JSON must be surfaced as an
			// error, not silently unmarshalled into an empty "success".
			name: "error status with valid json body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`[]`))
			},
			wantErr:     true,
			errContains: "non-success status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/profiles/delegations/my", tt.handler)
			}

			_, output, err := GetMyDelegations(context.Background(), newReq(), GetMyDelegationsInput{})

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

func TestGetMyDelegationsNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/profiles/delegations/my", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetMyDelegations(context.Background(), newReq(), GetMyDelegationsInput{})
	})
}
