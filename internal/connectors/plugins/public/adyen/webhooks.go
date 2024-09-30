package adyen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/url"
	"strings"

	"github.com/adyen/adyen-go-api-library/v7/src/webhook"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/stack/libs/go-libs/pointer"
)

type webhookConfig struct {
	urlPath string
	fn      func(context.Context, models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error)
}

var webhookConfigs map[string]webhookConfig

func (p Plugin) initWebhookConfig() {
	webhookConfigs = map[string]webhookConfig{
		"standard": {
			urlPath: "/standard",
			fn:      p.translateStandardWebhook,
		},
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) error {
	if req.WebhookBaseUrl == "" {
		return errors.New("STACK_PUBLIC_URL is not set")
	}

	standardConfig := webhookConfigs["standard"]

	url, err := url.JoinPath(req.WebhookBaseUrl, standardConfig.urlPath)
	if err != nil {
		return err
	}

	log.Println(url)

	return p.client.CreateWebhook(ctx, url, req.ConnectorID)
}

func (p Plugin) translateStandardWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if !p.client.VerifyWebhookBasicAuth(req.Webhook.BasicAuth) {
		return models.TranslateWebhookResponse{}, errors.New("invalid basic auth")
	}

	webhooks, err := p.client.TranslateWebhook(string(req.Webhook.Body))
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	responses := make([]models.WebhookResponse, 0, len(*webhooks.NotificationItems))
	for _, item := range *webhooks.NotificationItems {
		if !p.client.VerifyWebhookHMAC(item) {
			continue
		}

		var payment *models.PSPPayment
		var err error
		switch item.NotificationRequestItem.EventCode {
		case webhook.EventCodeAuthorisation:
			payment, err = p.handleAuthorisation(item.NotificationRequestItem)
		case webhook.EventCodeAuthorisationAdjustment:
			payment, err = p.handleAuthorisationAdjustment(item.NotificationRequestItem)
		case webhook.EventCodeCancellation:
			payment, err = p.handleCancellation(item.NotificationRequestItem)
		case webhook.EventCodeCapture:
			payment, err = p.handleCapture(item.NotificationRequestItem)
		case webhook.EventCodeCaptureFailed:
			payment, err = p.handleCaptureFailed(item.NotificationRequestItem)
		case webhook.EventCodeRefund:
			payment, err = p.handleRefund(item.NotificationRequestItem)
		case webhook.EventCodeRefundFailed:
			payment, err = p.handleRefundFailed(item.NotificationRequestItem)
		case webhook.EventCodeRefundedReversed:
			payment, err = p.handleRefundedReversed(item.NotificationRequestItem)
		case webhook.EventCodeRefundWithData:
			payment, err = p.handleRefundWithData(item.NotificationRequestItem)
		case webhook.EventCodePayoutThirdparty:
			payment, err = p.handlePayoutThirdparty(item.NotificationRequestItem)
		case webhook.EventCodePayoutDecline:
			payment, err = p.handlePayoutDecline(item.NotificationRequestItem)
		case webhook.EventCodePayoutExpire:
			payment, err = p.handlePayoutExpire(item.NotificationRequestItem)
		}
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		if payment != nil {
			responses = append(responses, models.WebhookResponse{
				IdempotencyKey: fmt.Sprintf("%s-%s-%d", item.NotificationRequestItem.MerchantReference, item.NotificationRequestItem.EventCode, item.NotificationRequestItem.EventDate.UnixNano()),
				Payment:        payment,
			})
		}
	}

	return models.TranslateWebhookResponse{
		Responses: responses,
	}, nil
}

func (p Plugin) handleAuthorisation(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	status := models.PAYMENT_STATUS_AUTHORISATION
	if item.Success == "false" {
		status = models.PAYMENT_STATUS_FAILED
	}

	payment := models.PSPPayment{
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      parseScheme(item.PaymentMethod),
		Status:                      status,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}

	return &payment, nil
}

func (p Plugin) handleAuthorisationAdjustment(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	payment := models.PSPPayment{
		ParentReference:             item.OriginalReference,
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_AMOUNT_ADJUSTEMENT,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}
	return &payment, nil
}

func (p Plugin) handleCancellation(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	payment := models.PSPPayment{
		ParentReference:             item.OriginalReference,
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      parseScheme(item.PaymentMethod),
		Status:                      models.PAYMENT_STATUS_CANCELLED,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}

	return &payment, nil
}

func (p Plugin) handleCapture(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	if item.Success == "false" {
		return nil, nil
	}

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	payment := models.PSPPayment{
		ParentReference:             item.OriginalReference,
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_CAPTURE,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}
	return &payment, nil
}

func (p Plugin) handleCaptureFailed(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	if item.Success == "false" {
		return nil, nil
	}

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	payment := models.PSPPayment{
		ParentReference:             item.OriginalReference,
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_CAPTURE_FAILED,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}

	return &payment, nil
}

func (p Plugin) handleRefund(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	if item.Success == "false" {
		return nil, nil
	}

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	payment := models.PSPPayment{
		ParentReference:             item.OriginalReference,
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_REFUNDED,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}

	return &payment, nil
}

func (p Plugin) handleRefundFailed(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	if item.Success == "false" {
		return nil, nil
	}

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	payment := models.PSPPayment{
		ParentReference:             item.OriginalReference,
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_REFUNDED_FAILURE,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}
	return &payment, nil
}

func (p Plugin) handleRefundedReversed(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	if item.Success == "false" {
		return nil, nil
	}

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	payment := models.PSPPayment{
		ParentReference:             item.OriginalReference,
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_REFUND_REVERSED,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}

	return &payment, nil
}

func (p Plugin) handleRefundWithData(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	if item.Success == "false" {
		return nil, nil
	}

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	payment := models.PSPPayment{
		ParentReference:             item.OriginalReference,
		Reference:                   item.PspReference,
		CreatedAt:                   *item.EventDate,
		Type:                        models.PAYMENT_TYPE_PAYIN,
		Amount:                      big.NewInt(item.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_REFUNDED,
		DestinationAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                         raw,
	}

	return &payment, nil
}

func (p Plugin) handlePayoutThirdparty(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	status := models.PAYMENT_STATUS_SUCCEEDED
	if item.Success == "false" {
		status = models.PAYMENT_STATUS_FAILED
	}

	payment := models.PSPPayment{
		Reference:              item.PspReference,
		CreatedAt:              *item.EventDate,
		Type:                   models.PAYMENT_TYPE_PAYOUT,
		Amount:                 big.NewInt(item.Amount.Value),
		Asset:                  currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                 models.PAYMENT_SCHEME_OTHER,
		Status:                 status,
		SourceAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                    raw,
	}

	return &payment, nil
}

func (p Plugin) handlePayoutDecline(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	if item.Success == "false" {
		return nil, nil
	}

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	payment := models.PSPPayment{
		ParentReference:        item.OriginalReference,
		Reference:              item.PspReference,
		CreatedAt:              *item.EventDate,
		Type:                   models.PAYMENT_TYPE_PAYOUT,
		Amount:                 big.NewInt(item.Amount.Value),
		Asset:                  currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                 models.PAYMENT_SCHEME_OTHER,
		Status:                 models.PAYMENT_STATUS_FAILED,
		SourceAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                    raw,
	}

	return &payment, nil
}

func (p Plugin) handlePayoutExpire(
	item webhook.NotificationRequestItem,
) (*models.PSPPayment, error) {
	if item.Success == "false" {
		return nil, nil
	}

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	payment := models.PSPPayment{
		ParentReference:        item.OriginalReference,
		Reference:              item.PspReference,
		CreatedAt:              *item.EventDate,
		Type:                   models.PAYMENT_TYPE_PAYOUT,
		Amount:                 big.NewInt(item.Amount.Value),
		Asset:                  currency.FormatAsset(supportedCurrenciesWithDecimal, item.Amount.Currency),
		Scheme:                 models.PAYMENT_SCHEME_OTHER,
		Status:                 models.PAYMENT_STATUS_EXPIRED,
		SourceAccountReference: pointer.For(item.MerchantAccountCode),
		Raw:                    raw,
	}

	return &payment, nil
}

func parseScheme(scheme string) models.PaymentScheme {
	switch {
	case strings.HasPrefix(scheme, "visa"):
		return models.PAYMENT_SCHEME_CARD_VISA
	case strings.HasPrefix(scheme, "electron"):
		return models.PAYMENT_SCHEME_CARD_VISA
	case strings.HasPrefix(scheme, "amex"):
		return models.PAYMENT_SCHEME_CARD_AMEX
	case strings.HasPrefix(scheme, "alipay"):
		return models.PAYMENT_SCHEME_CARD_ALIPAY
	case strings.HasPrefix(scheme, "cup"):
		return models.PAYMENT_SCHEME_CARD_CUP
	case strings.HasPrefix(scheme, "discover"):
		return models.PAYMENT_SCHEME_CARD_DISCOVER
	case strings.HasPrefix(scheme, "doku"):
		return models.PAYMENT_SCHEME_DOKU
	case strings.HasPrefix(scheme, "dragonpay"):
		return models.PAYMENT_SCHEME_DRAGON_PAY
	case strings.HasPrefix(scheme, "jcb"):
		return models.PAYMENT_SCHEME_CARD_JCB
	case strings.HasPrefix(scheme, "maestro"):
		return models.PAYMENT_SCHEME_MAESTRO
	case strings.HasPrefix(scheme, "mc"):
		return models.PAYMENT_SCHEME_CARD_MASTERCARD
	case strings.HasPrefix(scheme, "molpay"):
		return models.PAYMENT_SCHEME_MOL_PAY
	case strings.HasPrefix(scheme, "diners"):
		return models.PAYMENT_SCHEME_CARD_DINERS
	default:
		return models.PAYMENT_SCHEME_OTHER
	}
}
