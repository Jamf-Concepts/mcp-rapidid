// Copyright 2026, Jamf Software LLC

package ri

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
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

func ToolSetup(req *mcp.CallToolRequest, loggerName string) (*rapididentity.Client, *helper.ToolHelper, error) {
	th := helper.NewToolHelper(req, loggerName)
	options := GetRapidIdentityOptions()

	if options.RapidIdentityUser != nil && options.RapidIdentityUser.Username != "" {
		th.Logger().Debug(fmt.Sprintf("connecting to %s with user %s with a password of length %d", options.BaseUrl, options.RapidIdentityUser.Username, len(options.RapidIdentityUser.Password)))
	} else {
		th.Logger().Debug(fmt.Sprintf("connecting to %s with a service identity with key length %d", options.BaseUrl, len(options.ServiceIdentity)))
	}

	client, err := rapididentity.New(options)
	if err != nil {
		LogRIError(th, "unable to establish rapididentity connection", err)
		return nil, nil, err
	}

	return client, th, nil
}

func LogRIError(th *helper.ToolHelper, message string, err error) {
	riError, ok := err.(rapididentity.RapidIdentityError)
	if ok {
		th.Logger().Error(
			message,
			"error", riError.Message,
			"reason", riError.Reason,
			"method", riError.Method,
			"reqUrl", riError.ReqUrl.String(),
			"code", riError.Code)
	} else {
		th.Logger().Error(message, "error", err)
	}
}
