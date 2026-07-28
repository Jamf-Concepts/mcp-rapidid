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

func ToolSetup(ctx context.Context, req *mcp.CallToolRequest, toolName string) (context.Context, *rapididentity.Client, *helper.ToolHelper, error) {
	clientName, clientVersion := ClientInfoFromSession(req.Session)
	// StdioTransport does not assign session IDs — req.Session.ID() always returns "".
	// The SDK's GetSessionID option only applies to HTTP/SSE transports.
	// Passing "" here is intentional; if the SDK ever starts providing session IDs
	// for stdio, swapping this to req.Session.ID() will automatically enable grouping.
	ctx = telemetry.WithSession(ctx, req.Session.ID(), clientName, clientVersion)

	th := helper.NewToolHelper(req, toolName)
	options := GetRapidIdentityOptions()

	if options.RapidIdentityUser != nil && options.RapidIdentityUser.Username != "" {
		th.Logger().Debug(fmt.Sprintf("connecting to %s with user %s with a password of length %d", options.BaseUrl, options.RapidIdentityUser.Username, len(options.RapidIdentityUser.Password)))
	} else {
		th.Logger().Debug(fmt.Sprintf("connecting to %s with a service identity with key length %d", options.BaseUrl, len(options.ServiceIdentity)))
	}

	client, err := rapididentity.New(options)
	if err != nil {
		LogRIError(ctx, th, "unable to establish rapididentity connection", err)
		return ctx, nil, nil, err
	}

	return ctx, client, th, nil
}

func LogRIError(ctx context.Context, th *helper.ToolHelper, message string, err error) {
	riError, ok := err.(rapididentity.RapidIdentityError)
	if ok {
		th.Logger().Error(
			message,
			"error", riError.Message,
			"reason", riError.Reason,
			"method", riError.Method,
			"reqUrl", riError.ReqUrl.String(),
			"code", riError.Code)
		telemetry.RecordError(ctx,
			fmt.Sprintf(telemetry.EventPrefix+"ToolError.%s", th.ToolName()),
			fmt.Sprintf("error occurred calling RapidIdentity API with status code %d", riError.Code),
			telemetry.ErrorCategoryThrownException)
	} else {
		th.Logger().Error(message, "error", err)
		telemetry.RecordError(ctx,
			fmt.Sprintf(telemetry.EventPrefix+"ToolError.%s", th.ToolName()),
			"see server logs",
			telemetry.ErrorCategoryThrownException)
	}
}
