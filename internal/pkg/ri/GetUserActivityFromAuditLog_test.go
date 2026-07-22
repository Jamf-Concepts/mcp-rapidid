// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetUserActivityFromAuditLog(t *testing.T) {
	tests := []struct {
		name         string
		input        GetUserActivityFromAuditLogInput
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		assertOutput func(t *testing.T, output GetUserActivityFromAuditLogOutput)
	}{
		{
			name: "success with relative date range",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:  "user-123",
				DN:        "idauto=user-123,ou=Accounts,dc=meta",
				DateRange: "TODAY",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"auditLogRecords":[],"adminLimitEnforced":false,"nextPageToken":""}`))
			},
			assertOutput: func(t *testing.T, output GetUserActivityFromAuditLogOutput) {
				if len(output.AuditLogRecords) != 0 {
					t.Fatalf("expected empty records, got %d", len(output.AuditLogRecords))
				}
				if output.AdminLimitEnforced {
					t.Fatal("expected adminLimitEnforced to be false")
				}
			},
		},
		{
			name: "success with custom date range",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:    "user-123",
				DN:          "idauto=user-123,ou=Accounts,dc=meta",
				DateRange:   "CUSTOM",
				StartDate:   "07/01/2026 00:00:00",
				StartDateOp: "gt",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"auditLogRecords":[],"adminLimitEnforced":false,"nextPageToken":""}`))
			},
			assertOutput: func(t *testing.T, output GetUserActivityFromAuditLogOutput) {
				if len(output.AuditLogRecords) != 0 {
					t.Fatalf("expected empty records, got %d", len(output.AuditLogRecords))
				}
				if output.AdminLimitEnforced {
					t.Fatal("expected adminLimitEnforced to be false")
				}
			},
		},
		{
			name: "missing idautoId",
			input: GetUserActivityFromAuditLogInput{
				DN:        "idauto=user-123,ou=Accounts,dc=meta",
				DateRange: "TODAY",
			},
			wantErr:     true,
			errContains: "idautoId is required",
		},
		{
			name: "missing dn",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:  "user-123",
				DateRange: "TODAY",
			},
			wantErr:     true,
			errContains: "dn is required",
		},
		{
			name: "invalid date range",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:  "user-123",
				DN:        "idauto=user-123,ou=Accounts,dc=meta",
				DateRange: "INVALID",
			},
			wantErr:     true,
			errContains: "invalid dateRange",
		},
		{
			name: "custom date range missing startDate",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:  "user-123",
				DN:        "idauto=user-123,ou=Accounts,dc=meta",
				DateRange: "CUSTOM",
			},
			wantErr:     true,
			errContains: "startDate is required",
		},
		{
			name: "custom date range missing startDateOp",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:  "user-123",
				DN:        "idauto=user-123,ou=Accounts,dc=meta",
				DateRange: "CUSTOM",
				StartDate: "07/01/2026 00:00:00",
			},
			wantErr:     true,
			errContains: "startDateOp is required",
		},
		{
			name: "invalid startDateOp",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:    "user-123",
				DN:          "idauto=user-123,ou=Accounts,dc=meta",
				DateRange:   "CUSTOM",
				StartDate:   "07/01/2026 00:00:00",
				StartDateOp: "invalid",
			},
			wantErr:     true,
			errContains: "invalid startDateOp",
		},
		{
			name: "invalid startDate format",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:    "user-123",
				DN:          "idauto=user-123,ou=Accounts,dc=meta",
				DateRange:   "CUSTOM",
				StartDate:   "2026-07-01",
				StartDateOp: "gt",
			},
			wantErr:     true,
			errContains: "startDate must be in format",
		},
		{
			name: "custom date range missing endDateOp",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:    "user-123",
				DN:          "idauto=user-123,ou=Accounts,dc=meta",
				DateRange:   "CUSTOM",
				StartDate:   "07/01/2026 00:00:00",
				StartDateOp: "gt",
				EndDate:     "07/31/2026 00:00:00",
			},
			wantErr:     true,
			errContains: "endDateOp is required",
		},
		{
			name: "invalid endDateOp",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:    "user-123",
				DN:          "idauto=user-123,ou=Accounts,dc=meta",
				DateRange:   "CUSTOM",
				StartDate:   "07/01/2026 00:00:00",
				StartDateOp: "gt",
				EndDate:     "07/31/2026 00:00:00",
				EndDateOp:   "invalid",
			},
			wantErr:     true,
			errContains: "invalid endDateOp",
		},
		{
			name: "invalid endDate format",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:    "user-123",
				DN:          "idauto=user-123,ou=Accounts,dc=meta",
				DateRange:   "CUSTOM",
				StartDate:   "07/01/2026 00:00:00",
				StartDateOp: "gt",
				EndDate:     "2026-07-31",
				EndDateOp:   "lt",
			},
			wantErr:     true,
			errContains: "endDate must be in format",
		},
		{
			name: "malformed json body on 200",
			input: GetUserActivityFromAuditLogInput{
				IdautoID:  "user-123",
				DN:        "idauto=user-123,ou=Accounts,dc=meta",
				DateRange: "TODAY",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"auditLogRecords":[`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := setup(t)
			if tt.handler != nil {
				mux.HandleFunc(baseUrlPath+"/reporting/auditQuery", tt.handler)
			}

			_, output, err := GetUserActivityFromAuditLog(context.Background(), newReq(), tt.input)

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

func TestGetUserActivityFromAuditLogNoSecretLeak(t *testing.T) {
	mux := setup(t)
	mux.HandleFunc(baseUrlPath+"/reporting/auditQuery", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"auditLogRecords":[],"adminLimitEnforced":false,"nextPageToken":""}`))
	})

	assertNoSecretLeak(t, []string{mockPassword, "abcd"}, func() {
		GetUserActivityFromAuditLog(context.Background(), newReq(), GetUserActivityFromAuditLogInput{
			IdautoID:  "user-123",
			DN:        "idauto=user-123,ou=Accounts,dc=meta",
			DateRange: "TODAY",
		})
	})
}
