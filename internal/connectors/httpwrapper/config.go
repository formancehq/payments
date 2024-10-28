package httpwrapper

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/oauth2/clientcredentials"
)

type Config struct {
	HttpErrorCheckerFn      func(code int) error
	CommonMetricsAttributes []attribute.KeyValue

	Timeout     time.Duration
	Transport   http.RoundTripper
	OAuthConfig *clientcredentials.Config
}

func CommonMetricsAttributesFor(connectorName string) []attribute.KeyValue {
	metricsAttributes := []attribute.KeyValue{
		attribute.String("connector", connectorName),
	}
	return metricsAttributes
}
