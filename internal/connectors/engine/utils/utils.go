package utils

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/formancehq/payments/pkg/domain/models"
)

// FormanceRedirectURLQueryParamID must stay in sync with
// ce/plugins/plaid/client.FormanceRedirectURLQueryParamID - duplicated here
// (rather than imported) because that plugin lives in its own Go module
// and this package can't depend on a specific provider's implementation.
const FormanceRedirectURLQueryParamID = "formanceRedirectURL"

var ErrInvalidFormanceRedirectURL = errors.New("invalid formanceRedirectURL")

// ValidateFormanceRedirectURLQueryValues checks an incoming webhook's
// formanceRedirectURL query parameter, if present, against this app's own
// stackPublicURL before the webhook is handed off to any plugin.
//
// Unlike the webhook body (covered by the provider's own signature, e.g.
// Plaid's Plaid-Verification JWT), a query parameter on the delivery URL
// isn't cryptographically bound to anything - so a plugin that later uses
// this value as an outbound destination (see plaid's
// FormanceOpenBankingRedirect) must not trust it as-is. Requiring it to
// resolve to this app's own public origin closes that gap without needing
// per-plugin, per-caller validation logic.
func ValidateFormanceRedirectURLQueryValues(stackPublicURL string, queryValues map[string][]string) error {
	values, ok := queryValues[FormanceRedirectURLQueryParamID]
	if !ok {
		// Not every provider/session uses this convention - nothing to
		// validate.
		return nil
	}
	if len(values) != 1 {
		return fmt.Errorf("%w: expected exactly one value, got %d", ErrInvalidFormanceRedirectURL, len(values))
	}

	return validateFormanceRedirectURL(stackPublicURL, values[0])
}

func validateFormanceRedirectURL(stackPublicURL, redirectURL string) error {
	base, err := url.Parse(stackPublicURL)
	if err != nil {
		return fmt.Errorf("parsing stack public URL: %w", err)
	}

	got, err := url.Parse(redirectURL)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrInvalidFormanceRedirectURL, redirectURL, err)
	}

	if got.Scheme != base.Scheme || got.Host != base.Host {
		return fmt.Errorf("%w: %q does not match this app's public URL", ErrInvalidFormanceRedirectURL, redirectURL)
	}

	return nil
}

func GetWebhookBaseURL(stackPublicURL string, connectorID models.ConnectorID) (string, error) {
	webhookBaseURL, err := url.JoinPath(stackPublicURL, "api/payments/v3/connectors/webhooks", connectorID.String())
	if err != nil {
		return "", fmt.Errorf("joining webhook base URL: %w", err)
	}

	return webhookBaseURL, nil
}

func GetFormanceRedirectURL(stackPublicURL string, connectorID models.ConnectorID) (string, error) {
	formanceRedirectURL, err := url.JoinPath(stackPublicURL, "api/payments/v3/connectors/open-banking", connectorID.String(), "redirect")
	if err != nil {
		return "", fmt.Errorf("joining webhook base URL: %w", err)
	}

	return formanceRedirectURL, nil
}
