package prompts

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const passwordResetAuditPrompt = `
Look up {{.Username}} in RapidIdentity and check their audit log for the last 30 days for any password reset events.

	1. Resolve the user with mcp-rapidid:search-users.
	2. Fetch their audit log using mcp-rapidid:get-user-activity-from-audit-log with LAST_30_DAYS
	3. Filter the events to an action displayName of "Password Reset", "Self Password Update", "Change Password", and "Change Account Password" 
	4. For each reset, resolve the perpetrator's display name, if present:
		a. Try mcp-rapidid:search-users with the perpetratorId UUID.
        b. If that returns empty, call mcp-rapidid:get-user-info-in-delegation with {{.Delegation}} with LDAP filter (idautoID=<perpetratorId>)
		b. If {{.Delegation}} is empty, call mcp-rapidid:get-my-delegations, if you haven't already, to retrieve the available delegations and show these to the user asking "What delegation shoud I use to do the look up of the perpetrator", then use mcp-rapidid:get-user-info-in-delegation with LDAP filter (idautoID=<perpetratorId>) on the user selected delegation. if the user can still not be found simply use the UUID for the Change By column.
	5. The delegation used for the reset is in extendedProperties where key=delegationName.
	6. Whether the user must update their password on next login is in extendedProperties where key=mustUpdate.

If {{.Username}} is not provided, ask: "What is the full name or username of the user you would like to look up?" before proceeding. All timestamps must be in {{.Timezone}} if provided, otherwise use the user's local timezone.

Output ONLY the table — no other text — with columns: Timestamp, User, Changed By, Delegation Used, Successful, Must Update on Next Login.
Sort the table by timestamp from most recent to least recent.
`

type PasswordRestAuditArgs struct {
	Username   string
	Timezone   string
	Delegation string
}

const passwordResetAuditPromptName = "password-reset-audit"

func PasswordResetAuditPromptDef() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        passwordResetAuditPromptName,
		Description: "Finds all password resets for the specified user in the last 30 days and provides a table that includes the date the password was changed, who it was changed by, what delegation was used, whether it was successful, and whether the user was forced to change their password at next login",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "Username",
				Description: "The username, first name, or last name of the user",
				Required:    true,
			},
			{
				Name:        "Timezone",
				Description: "The timezone to display dates and times",
				Required:    false,
			},
			{
				Name:        "Delegation",
				Description: "The delegation to search for users if the Global search comes back empty",
				Required:    false,
			},
		},
	}
}

func PasswordResetAuditPromptHandler(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	var err error
	ctx, sh, setupErr := PromptSetup(ctx, req, passwordResetAuditPromptName)
	if setupErr != nil {
		return nil, setupErr
	}
	start := time.Now()
	defer telemetry.RecordPrompt(ctx, passwordResetAuditPromptName, start, &err)

	sh.Logger().Info(passwordResetAuditPromptName + " prompt used")

	promptArgs := PasswordRestAuditArgs{
		Username:   req.Params.Arguments["Username"],
		Timezone:   req.Params.Arguments["Timezone"],
		Delegation: req.Params.Arguments["Delegation"],
	}

	tmpl, err := template.New("Password Reset Audit Prompt").Parse(passwordResetAuditPrompt)
	if err != nil {
		sh.Logger().Error("unable to instantiate template for prompt "+passwordResetAuditPromptName, "error", err)
		telemetry.RecordError(ctx,
			fmt.Sprintf(telemetry.EventPrefix+telemetry.PromptErrorPrefix+"%s", sh.PromptName()),
			err.Error(),
			telemetry.ErrorCategoryThrownException)
		return nil, err
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, promptArgs)
	if err != nil {
		sh.Logger().Error("unable to execute template for prompt "+passwordResetAuditPromptName, "error", err)
		telemetry.RecordError(ctx,
			fmt.Sprintf(telemetry.EventPrefix+telemetry.PromptErrorPrefix+"%s", sh.PromptName()),
			err.Error(),
			telemetry.ErrorCategoryThrownException)
		return nil, err
	}

	prompt := buf.String()
	sh.Logger().Debug("The prompt being used", "prompt", prompt)

	return &mcp.GetPromptResult{
		Description: "Password Reset Audit Prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: prompt},
			},
		},
	}, nil
}
