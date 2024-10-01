package client

import (
	"net/http"
	"strings"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/hashicorp/go-hclog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	httpClient httpwrapper.Client
	endpoint   string
}

func New(logger hclog.Logger, clientID, apiKey, endpoint string) (*Client, error) {
	config := &httpwrapper.Config{
		Transport: &apiTransport{
			logger:     logger,
			clientID:   clientID,
			apiKey:     apiKey,
			endpoint:   endpoint,
			underlying: otelhttp.NewTransport(http.DefaultTransport),
		},
		HttpErrorCheckerFn: func(statusCode int) error {
			if statusCode == http.StatusNotFound {
				return nil
			}
			if statusCode >= http.StatusBadRequest {
				return httpwrapper.ErrStatusCodeUnexpected
			}
			return nil

		},
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	httpClient, err := httpwrapper.NewClient(logger, config)
	c := &Client{
		httpClient: httpClient,
		endpoint:   endpoint,
	}
	return c, err
}
