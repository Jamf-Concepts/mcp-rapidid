package ri

import (
	"net/http"
	"net/url"
	"os"

	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

func GetRapidIdentityOptions() rapididentity.Options {
	options := rapididentity.Options{
		HTTPClient: &http.Client{},
		BaseUrl:    &url.URL{Scheme: "https", Host: os.Getenv("RI_HOST")},
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
