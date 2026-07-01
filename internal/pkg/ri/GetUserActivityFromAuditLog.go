// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetUserActivityFromAuditLogInput struct {
	IdautoID    string `json:"idautoId"    jsonschema:"required,The unique RapidIdentity user identifier (idautoID UUID). Use search-users to resolve a name first."`
	DN          string `json:"dn"          jsonschema:"required,The distinguished name (DN) of the user (e.g. idautoID=<uuid>,ou=Accounts,dc=meta). Use search-users to resolve a name first."`
	DateRange   string `json:"dateRange"   jsonschema:"required,Relative: TODAY YESTERDAY LAST_7_DAYS LAST_30_DAYS LAST_12_MONTHS THIS_WEEK THIS_MONTH THIS_YEAR. Or CUSTOM (requires startDate and startDateOp)."`
	StartDate   string `json:"startDate"   jsonschema:"A custom date in UTC. Required when dateRange is CUSTOM. Format: MM/DD/YYYY HH:MM:SS"`
	StartDateOp string `json:"startDateOp" jsonschema:"Operator for startDate. Required when startDate is set. Allowed values: gt, lt, eq"`
	EndDate     string `json:"endDate"     jsonschema:"An optional second custom date in UTC. Format: MM/DD/YYYY HH:MM:SS"`
	EndDateOp   string `json:"endDateOp"   jsonschema:"Operator for endDate. Required when endDate is set. Allowed values: gt, lt, eq"`
	PageSize    int    `json:"pageSize"    jsonschema:"Maximum number of records to return per page. Omit or set to 0 to use the server default."`
	PageToken   string `json:"pageToken"   jsonschema:"Token from a previous response's nextPageToken to retrieve the next page of results."`
}

type GetUserActivityFromAuditLogOutput struct {
	AuditLogRecords    []rapididentity.AuditReportResult `json:"auditLogRecords"    jsonschema:"Audit log entries returned for the user"`
	AdminLimitEnforced bool                              `json:"adminLimitEnforced" jsonschema:"True if the admin result limit was reached"`
	NextPageToken      string                            `json:"nextPageToken"      jsonschema:"Token to pass as pageToken to retrieve the next page. Empty when there are no more pages."`
}

var validRelativeDateRanges = []string{
	"TODAY",
	"YESTERDAY",
	"LAST_7_DAYS",
	"LAST_30_DAYS",
	"LAST_12_MONTHS",
	"THIS_WEEK",
	"THIS_MONTH",
	"THIS_YEAR",
}

var validDateOperators = []string{"gt", "lt", "eq"}

const auditTimestampFormat = "01/02/2006 15:04:05"

func isValidRelativeDateRange(dateRange string) bool {
	for _, v := range validRelativeDateRanges {
		if v == dateRange {
			return true
		}
	}
	return false
}

func isValidDateOperator(op string) bool {
	for _, v := range validDateOperators {
		if v == op {
			return true
		}
	}
	return false
}

func GetUserActivityFromAuditLog(ctx context.Context, req *mcp.CallToolRequest, input GetUserActivityFromAuditLogInput) (*mcp.CallToolResult, GetUserActivityFromAuditLogOutput, error) {
	empty := GetUserActivityFromAuditLogOutput{}

	if input.IdautoID == "" {
		return nil, empty, fmt.Errorf("idautoId is required")
	}
	if input.DN == "" {
		return nil, empty, fmt.Errorf("dn is required")
	}

	isCustom := input.DateRange == "CUSTOM"
	if !isCustom && !isValidRelativeDateRange(input.DateRange) {
		return nil, empty, fmt.Errorf("invalid dateRange %q: must be a relative value (TODAY, YESTERDAY, LAST_7_DAYS, LAST_30_DAYS, LAST_12_MONTHS, THIS_WEEK, THIS_MONTH, THIS_YEAR) or CUSTOM", input.DateRange)
	}

	if isCustom {
		if input.StartDate == "" {
			return nil, empty, fmt.Errorf("startDate is required when dateRange is CUSTOM")
		}
		if input.StartDateOp == "" {
			return nil, empty, fmt.Errorf("startDateOp is required when startDate is set")
		}
		if !isValidDateOperator(input.StartDateOp) {
			return nil, empty, fmt.Errorf("invalid startDateOp %q: must be gt, lt, or eq", input.StartDateOp)
		}
		if _, err := time.Parse(auditTimestampFormat, input.StartDate); err != nil {
			return nil, empty, fmt.Errorf("startDate must be in format MM/DD/YYYY HH:MM:SS: %w", err)
		}
		if input.EndDate != "" {
			if input.EndDateOp == "" {
				return nil, empty, fmt.Errorf("endDateOp is required when endDate is set")
			}
			if !isValidDateOperator(input.EndDateOp) {
				return nil, empty, fmt.Errorf("invalid endDateOp %q: must be gt, lt, or eq", input.EndDateOp)
			}
			if _, err := time.Parse(auditTimestampFormat, input.EndDate); err != nil {
				return nil, empty, fmt.Errorf("endDate must be in format MM/DD/YYYY HH:MM:SS: %w", err)
			}
		}
	}

	options := GetRapidIdentityOptions()
	client, err := rapididentity.New(options)
	if err != nil {
		return nil, empty, err
	}
	defer func(c *rapididentity.Client) {
		if cerr := c.Close(); cerr != nil {
			_, _ = fmt.Fprint(os.Stderr, cerr)
		}
	}(client)

	targetNode := rapididentity.AuditReportQuery{
		FieldName:          "target",
		FieldSecondaryName: "targetId",
		OperatorType:       rapididentity.EQUAL,
		FieldValues: []rapididentity.AuditReportFieldValue{
			{Dn: input.DN, FieldNameAndServerId: "Person", Id: input.IdautoID},
		},
	}

	childNodes := []rapididentity.AuditReportQuery{targetNode}

	if isCustom {
		childNodes = append(childNodes, rapididentity.AuditReportQuery{
			FieldName:          "timestamp",
			FieldSecondaryName: "timestamp",
			OperatorType:       rapididentity.AuditReportOperator(input.StartDateOp),
			FieldValue:         input.StartDate,
		})
		if input.EndDate != "" {
			childNodes = append(childNodes, rapididentity.AuditReportQuery{
				FieldName:          "timestamp",
				FieldSecondaryName: "timestamp",
				OperatorType:       rapididentity.AuditReportOperator(input.EndDateOp),
				FieldValue:         input.EndDate,
			})
		}
	} else {
		childNodes = append(childNodes, rapididentity.AuditReportQuery{
			FieldName:          "timestamp",
			FieldSecondaryName: "timestamp",
			OperatorType:       rapididentity.EQUAL,
			FieldValues: []rapididentity.AuditReportFieldValue{
				{Dn: input.DateRange, FieldNameAndServerId: input.DateRange, Id: input.DateRange, Name: input.DateRange},
			},
		})
	}

	result, err := client.RunAuditReport(ctx, rapididentity.RunAuditReportInput{
		Query: rapididentity.AuditReportQuery{
			OperatorType: rapididentity.AND,
			ChildNodes:   childNodes,
		},
		PageSize:  input.PageSize,
		PageToken: input.PageToken,
	})
	if err != nil {
		return nil, empty, err
	}

	return nil, GetUserActivityFromAuditLogOutput{
		AuditLogRecords:    result.AuditLogRecords,
		AdminLimitEnforced: result.AdminLimitEnforced,
		NextPageToken:      result.NextPageToken,
	}, nil
}
