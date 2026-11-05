// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AuthMode identifies which kind of RapidIdentity session a tool call uses.
type AuthMode string

const (
	AuthModeServiceIdentity AuthMode = "service-identity"
	AuthModeUserPassword    AuthMode = "user-password"
)

// serviceIdentityUnsupportedTools lists tools whose underlying RapidIdentity
// endpoints do not work with a Service Identity. Only add a tool here once it
// has been verified against a live tenant: unsupported endpoints frequently
// answer a Service Identity with HTTP 200 and an empty body rather than a
// 401/403, so the failure cannot be detected from the response.
var serviceIdentityUnsupportedTools = map[string]bool{
	getUserInfoInDelegationToolName: true,
}

// ToolSupportsServiceIdentity reports whether the named tool can be called with
// a Service Identity session.
func ToolSupportsServiceIdentity(toolName string) bool {
	return !serviceIdentityUnsupportedTools[toolName]
}

// ServiceIdentityUnsupportedError is returned when only a Service Identity is
// configured and the requested tool cannot be used with one.
type ServiceIdentityUnsupportedError struct {
	ToolName string
}

func (e ServiceIdentityUnsupportedError) Error() string {
	return fmt.Sprintf("the %s tool is not supported with a RapidIdentity Service Identity; configure RI_USER and RI_PASSWORD to use it", e.ToolName)
}

func serviceIdentityConfigured() bool {
	return os.Getenv("RI_SERVICE_IDENTITY_SECRET_KEY") != ""
}

func userCredentialsConfigured() bool {
	return os.Getenv("RI_USER") != ""
}

// AuthModeForTool selects the session type for a tool call. A Service Identity
// is preferred when the tool supports it; otherwise username/password is used
// when configured. If only a Service Identity is configured and the tool does
// not support it, a ServiceIdentityUnsupportedError is returned.
func AuthModeForTool(toolName string) (AuthMode, error) {
	sid := serviceIdentityConfigured()
	switch {
	case sid && ToolSupportsServiceIdentity(toolName):
		return AuthModeServiceIdentity, nil
	case userCredentialsConfigured():
		return AuthModeUserPassword, nil
	case sid:
		return "", ServiceIdentityUnsupportedError{ToolName: toolName}
	default:
		// Nothing usable is configured; attempt username/password so the
		// resulting authentication failure is surfaced to the caller.
		return AuthModeUserPassword, nil
	}
}

func baseRapidIdentityOptions() rapididentity.Options {
	host := os.Getenv("RI_HOST")
	scheme := "https"
	if strings.HasPrefix(host, "http://") {
		scheme = "http"
	}
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")

	return rapididentity.Options{
		HTTPClient: &http.Client{},
		BaseUrl:    &url.URL{Scheme: scheme, Host: host},
	}
}

func rapidIdentityOptionsFor(mode AuthMode) rapididentity.Options {
	options := baseRapidIdentityOptions()
	if mode == AuthModeServiceIdentity {
		options.ServiceIdentity = os.Getenv("RI_SERVICE_IDENTITY_SECRET_KEY")
	} else {
		options.RapidIdentityUser = &rapididentity.RapidIdentityUser{
			Username: os.Getenv("RI_USER"),
			Password: os.Getenv("RI_PASSWORD"),
		}
	}
	return options
}

// GetRapidIdentityOptions returns client options without regard to a specific
// tool. A Service Identity takes precedence over username/password.
func GetRapidIdentityOptions() rapididentity.Options {
	if serviceIdentityConfigured() {
		return rapidIdentityOptionsFor(AuthModeServiceIdentity)
	}
	return rapidIdentityOptionsFor(AuthModeUserPassword)
}

// GetRapidIdentityOptionsForTool returns client options for the session type
// selected by AuthModeForTool.
func GetRapidIdentityOptionsForTool(toolName string) (rapididentity.Options, AuthMode, error) {
	mode, err := AuthModeForTool(toolName)
	if err != nil {
		return rapididentity.Options{}, "", err
	}
	return rapidIdentityOptionsFor(mode), mode, nil
}

type licenseInfo struct {
	LicenseeId string `json:"licenseeId"`
}

// GetLicenseeID retrieves the tenant's licensee ID from the RapidIdentity API.
// Used at startup to populate rimcpctx for telemetry.
func GetLicenseeID(ctx context.Context) (string, error) {
	options := GetRapidIdentityOptions()
	client, err := rapididentity.New(options)
	if err != nil {
		return "", err
	}
	defer client.Close()

	res, err := client.DoCustomRequest(ctx, "GET", "license/validated", nil)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var info licenseInfo
	if err = json.Unmarshal(body, &info); err != nil {
		return "", err
	}

	return info.LicenseeId, nil
}

// ClientInfoFromSession extracts the MCP client name and version from a session.
// Returns "unknown" for either field when the client does not provide them.
func ClientInfoFromSession(session *mcp.ServerSession) (string, string) {
	clientName := "unknown"
	clientVersion := "unknown"
	p := session.InitializeParams()
	if p != nil && p.ClientInfo != nil {
		if p.ClientInfo.Name != "" {
			clientName = p.ClientInfo.Name
		}
		if p.ClientInfo.Version != "" {
			clientVersion = p.ClientInfo.Version
		}
	}
	return clientName, clientVersion
}

func ToolSetup(ctx context.Context, req *mcp.CallToolRequest, toolName string) (context.Context, *rapididentity.Client, *helper.ServerHelper, error) {
	clientName, clientVersion := ClientInfoFromSession(req.Session)
	// StdioTransport does not assign session IDs — req.Session.ID() always returns "".
	// The SDK's GetSessionID option only applies to HTTP/SSE transports.
	// Passing "" here is intentional; if the SDK ever starts providing session IDs
	// for stdio, swapping this to req.Session.ID() will automatically enable grouping.
	ctx = telemetry.WithSession(ctx, req.Session.ID(), clientName, clientVersion)

	sh := helper.NewToolServerHelper(req, toolName)
	options, mode, err := GetRapidIdentityOptionsForTool(toolName)
	if err != nil {
		LogRIError(ctx, sh, "no supported credentials configured for tool", err)
		return ctx, nil, nil, err
	}

	if mode == AuthModeServiceIdentity {
		sh.Logger().Debug(fmt.Sprintf("connecting to %s with a service identity with key length %d", options.BaseUrl, len(options.ServiceIdentity)))
	} else {
		if serviceIdentityConfigured() {
			sh.Logger().Debug("tool does not support service identities; falling back to username/password")
		}
		sh.Logger().Debug(fmt.Sprintf("connecting to %s with user %s with a password of length %d", options.BaseUrl, options.RapidIdentityUser.Username, len(options.RapidIdentityUser.Password)))
	}

	client, err := rapididentity.New(options)
	if err != nil {
		LogRIError(ctx, sh, "unable to establish rapididentity connection", err)
		return ctx, nil, nil, err
	}

	return ctx, client, sh, nil
}

func LogRIError(ctx context.Context, sh *helper.ServerHelper, message string, err error) {
	riError, ok := err.(rapididentity.RapidIdentityError)
	if ok {
		sh.Logger().Error(
			message,
			"error", riError.Message,
			"reason", riError.Reason,
			"method", riError.Method,
			"reqUrl", riError.ReqUrl.String(),
			"code", riError.Code)
		telemetry.RecordError(ctx,
			fmt.Sprintf(telemetry.EventPrefix+"ToolError.%s", sh.ToolName()),
			fmt.Sprintf("error occurred calling RapidIdentity API with status code %d", riError.Code),
			telemetry.ErrorCategoryThrownException)
	} else {
		sh.Logger().Error(message, "error", err)
		telemetry.RecordError(ctx,
			fmt.Sprintf(telemetry.EventPrefix+"ToolError.%s", sh.ToolName()),
			"see server logs",
			telemetry.ErrorCategoryThrownException)
	}
}

// checkResponseStatus returns an error when an HTTP response is not a 2xx
// success. RapidIdentity's DoCustomRequest returns the raw response without
// validating the status code, so callers must guard against error responses
// before reading and unmarshalling the body; otherwise a 4xx/5xx is silently
// treated as an empty success.
func checkResponseStatus(res *http.Response) error {
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("rapididentity API returned non-success status %d", res.StatusCode)
	}
	return nil
}
