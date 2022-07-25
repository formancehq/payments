package hmac

import (
	"time"

	"github.com/google/uuid"
	"github.com/numary/payments/pkg/bridge/connectors/modulr/hmac/signature"
)

const (
	AuthorizationHeader = "Authorization"
	DateHeader          = "Date"
	EmptyString         = ""
	NonceHeader         = "x-mod-nonce"
	Retry               = "x-mod-retry"
	RetryTrue           = "true"
	RetryFalse          = "false"
)

var dateNow = time.Now

func GenerateHeaders(apiKey string, apiSecret string, nonce string, hasRetry bool) (map[string]string, *ValidationError) {
	validationError := validateInput(apiKey, apiSecret)

	if validationError != nil {
		return nil, validationError
	}
	return constructHeadersMap(apiKey, apiSecret, nonce, hasRetry), nil
}

func constructHeadersMap(apiKey string, apiSecret string, nonce string, hasRetry bool) map[string]string {
	headers := make(map[string]string)
	date := dateNow().Format(time.RFC1123)
	nonce = generateNonceIfEmpty(nonce)

	headers[DateHeader] = date
	headers[AuthorizationHeader] = signature.Build(apiKey, apiSecret, nonce, date)
	headers[NonceHeader] = nonce
	headers[Retry] = parseRetryBool(hasRetry)
	return headers
}

func generateNonceIfEmpty(nonce string) string {
	if nonce == EmptyString {
		nonce = uuid.New().String()
	}
	return nonce
}

func parseRetryBool(hasRetry bool) string {
	if hasRetry {
		return RetryTrue
	}
	return RetryFalse
}
