package utils

import (
	"fmt"
	"net/url"

	"github.com/formancehq/payments/internal/models"
)

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
