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

func GetRapidIdentityOptions() rapididentity.Options {
	host := os.Getenv("RI_HOST")
	scheme := "https"
	if strings.HasPrefix(host, "http://") {
		scheme = "http"
	}
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")

	options := rapididentity.Options{
		HTTPClient: &http.Client{},
		BaseUrl:    &url.URL{Scheme: scheme, Host: host},
	}

	serviceIdentitySecretKey := os.Getenv("RI_SERVICE_IDENTITY_SECRET_KEY")

	if serviceIdentitySecretKey == "" {
		options.RapidIdentityUser = &rapididentity.RapidIdentityUser{
			Username: os.Getenv("RI_USER"),
			Password: os.Getenv("RI_PASSWORD"),
		}
	} else {
		options.ServiceIdentity = serviceIdentitySecretKey
	}

	return options
}

// UsingServiceIdentity reports whether the server is configured to authenticate
// with a Service Identity. A Service Identity takes precedence over
// username/password, mirroring the selection made in GetRapidIdentityOptions.
func UsingServiceIdentity() bool {
	return os.Getenv("RI_SERVICE_IDENTITY_SECRET_KEY") != ""
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
	options := GetRapidIdentityOptions()

	if options.RapidIdentityUser != nil && options.RapidIdentityUser.Username != "" {
		sh.Logger().Debug(fmt.Sprintf("connecting to %s with user %s with a password of length %d", options.BaseUrl, options.RapidIdentityUser.Username, len(options.RapidIdentityUser.Password)))
	} else {
		sh.Logger().Debug(fmt.Sprintf("connecting to %s with a service identity with key length %d", options.BaseUrl, len(options.ServiceIdentity)))
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
