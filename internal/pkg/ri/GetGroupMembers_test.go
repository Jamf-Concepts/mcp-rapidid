// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetGroupMembers(t *testing.T) {
	tests := []struct {
		name         string
		input        GetGroupMembersInput
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output GetGroupMembersOutput)
	}{
		{
			name:  "success empty members",
			input: GetGroupMembersInput{GroupId: "group-123", PageSize: 1000},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"pagingSessionId":"","calculatedMembership":[],"totalCount":0}`))
			},
			assertOutput: func(t *testing.T, output GetGroupMembersOutput) {
				if len(output.CalculatedMembership) != 0 {
					t.Fatalf("expected empty membership, got %d", len(output.CalculatedMembership))
				}
				if output.TotalCount != 0 {
					t.Fatalf("expected totalCount 0, got %d", output.TotalCount)
				}
			},
		},
		{
			name:  "success with paging session id",
			input: GetGroupMembersInput{GroupId: "group-123", PageSize: 1000, PagingSessionId: "session-abc"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("pagingSessionId") != "session-abc" {
					t.Errorf("expected pagingSessionId=session-abc in request URL, got %q", r.URL.RawQuery)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"pagingSessionId":"session-def","calculatedMembership":[],"totalCount":5}`))
			},
			assertOutput: func(t *testing.T, output GetGroupMembersOutput) {
				if output.PagingSessionId != "session-def" {
					t.Fatalf("expected pagingSessionId session-def, got %q", output.PagingSessionId)
				}
				if output.TotalCount != 5 {
					t.Fatalf("expected totalCount 5, got %d", output.TotalCount)
				}
			},
		},
		{
			name:  "malformed json body on 200",
			input: GetGroupMembersInput{GroupId: "group-123", PageSize: 1000},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"calculatedMembership":[`))
			},
			wantErr: true,
		},
		{
			name:  "server error",
			input: GetGroupMembersInput{GroupId: "group-123", PageSize: 1000},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`not json`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/roles/groups/group-123/membershipCalculation", tt.handler)
			}

			_, output, err := GetGroupMembers(context.Background(), newReq(), tt.input)

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

func TestGetGroupMembersNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/roles/groups/group-123/membershipCalculation", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"pagingSessionId":"","calculatedMembership":[],"totalCount":0}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetGroupMembers(context.Background(), newReq(), GetGroupMembersInput{
			GroupId:  "group-123",
			PageSize: 1000,
		})
	})
}
