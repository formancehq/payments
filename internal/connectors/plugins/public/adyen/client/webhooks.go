package client

import (
	"context"

	"github.com/adyen/adyen-go-api-library/v7/src/hmacvalidator"
	"github.com/adyen/adyen-go-api-library/v7/src/management"
	"github.com/adyen/adyen-go-api-library/v7/src/webhook"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
)

func (c *client) searchWebhook(ctx context.Context, connectorID string) error {
	pageSize := 50
	for page := 1; ; page++ {
		webhooks, raw, err := c.client.Management().WebhooksCompanyLevelApi.ListAllWebhooks(
			metrics.OperationContext(ctx, "list_hooks"),
			c.client.Management().WebhooksCompanyLevelApi.ListAllWebhooksInput(c.companyID).PageNumber(int32(page)).PageSize(int32(pageSize)),
		)
		if err != nil {
			return c.wrapSDKError(err, raw.StatusCode)
		}

		if len(webhooks.Data) == 0 {
			break
		}

		for _, webhook := range webhooks.Data {
			if webhook.Description == nil {
				continue
			}

			if *webhook.Description != connectorID {
				continue
			}

			if webhook.Type != "standard" {
				continue
			}

			c.standardWebhook = &webhook
			break
		}

		if len(webhooks.Data) < pageSize {
			break
		}
	}

	return nil
}

func (c *client) CreateWebhook(ctx context.Context, url string, connectorID string) error {
	if c.standardWebhook != nil {
		return nil
	}

	if err := c.searchWebhook(ctx, connectorID); err != nil {
		return err
	}

	if c.standardWebhook != nil {
		return nil
	}

	req := management.CreateCompanyWebhookRequest{
		Active:                    true,
		CommunicationFormat:       "json",
		FilterMerchantAccountType: "allAccounts",
		Description:               pointer.For(connectorID),
		SslVersion:                pointer.For("TLSv1.3"),
		Type:                      "standard",
		Url:                       url,
	}

	if c.webhookUsername != "" {
		req.Username = pointer.For(c.webhookUsername)
	}

	if c.webhookPassword != "" {
		req.Password = pointer.For(c.webhookPassword)
	}

	webhook, raw, err := c.client.Management().WebhooksCompanyLevelApi.SetUpWebhook(
		metrics.OperationContext(ctx, "create_hook"),
		c.client.Management().WebhooksCompanyLevelApi.SetUpWebhookInput(c.companyID).
			CreateCompanyWebhookRequest(req),
	)
	if err != nil {
		return c.wrapSDKError(err, raw.StatusCode)
	}

	hmac, raw, err := c.client.Management().WebhooksCompanyLevelApi.GenerateHmacKey(
		metrics.OperationContext(ctx, "create_hook_hmac_key"),
		c.client.Management().WebhooksCompanyLevelApi.GenerateHmacKeyInput(c.companyID, *webhook.Id),
	)
	if err != nil {
		return c.wrapSDKError(err, raw.StatusCode)
	}

	c.standardWebhook = &webhook
	c.hmacKey = hmac.HmacKey

	return nil
}

func (c *client) VerifyWebhookBasicAuth(basicAuth *models.BasicAuth) bool {
	switch {
	case c.webhookUsername != "" && c.webhookPassword != "" && basicAuth == nil:
		return false
	case c.webhookUsername == "" && c.webhookPassword == "" && basicAuth == nil:
		return true
	case c.webhookUsername != "" && c.webhookPassword != "" && basicAuth != nil:
		return c.webhookUsername == basicAuth.Username && c.webhookPassword == basicAuth.Password
	}

	return false
}

func (c *client) VerifyWebhookHMAC(item webhook.NotificationItem) bool {
	return hmacvalidator.ValidateHmac(item.NotificationRequestItem, c.hmacKey)
}

func (c *client) DeleteWebhook(ctx context.Context, connectorID string) error {
	if c.standardWebhook == nil {
		if err := c.searchWebhook(ctx, connectorID); err != nil {
			return err
		}

		if c.standardWebhook == nil {
			return nil
		}
	}

	raw, err := c.client.Management().WebhooksCompanyLevelApi.RemoveWebhook(
		metrics.OperationContext(ctx, "delete_hook"),
		c.client.Management().WebhooksCompanyLevelApi.RemoveWebhookInput(c.companyID, *c.standardWebhook.Id),
	)
	if err != nil {
		return c.wrapSDKError(err, raw.StatusCode)
	}

	c.standardWebhook = nil
	return nil
}

func (c *client) TranslateWebhook(req string) (*webhook.Webhook, error) {
	return webhook.HandleRequest(req)
}
