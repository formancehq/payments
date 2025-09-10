package powens

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

const (
	webhookSecretMetadataKey = "secret"

	webhookAccountSyncedTransactionsLimit = 100
)

type supportedWebhook struct {
	urlPath        string
	trimFunction   func(context.Context, models.TrimWebhookRequest) (models.TrimWebhookResponse, error)
	handleFunction func(context.Context, models.TranslateWebhookRequest) ([]models.WebhookResponse, error)
}

func (p *Plugin) initWebhookConfig() {
	p.supportedWebhooks = map[client.WebhookEventType]supportedWebhook{
		client.WebhookEventTypeUserDeleted: {
			urlPath:        "/user-deleted",
			handleFunction: p.handleUserDeleted,
		},
		client.WebhookEventTypeConnectionSynced: {
			urlPath:        "/connection-synced",
			trimFunction:   p.trimConnectionSynced,
			handleFunction: p.handleConnectionSynced,
		},
		client.WebhookEventTypeConnectionDeleted: {
			urlPath:        "/connection-deleted",
			handleFunction: p.handleConnectionDeleted,
		},
		client.WebhookEventTypeAccountsFetched: {
			urlPath:        "/accounts-fetched",
			handleFunction: p.handleAccountsFetched,
		},
		client.WebhookEventTypeAccountSynced: {
			urlPath:        "/account-synced",
			trimFunction:   p.trimAccountsSynced,
			handleFunction: p.handleAccountSynced,
		},
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	secretKey, err := p.client.CreateWebhookAuth(ctx, p.name)
	if err != nil {
		return models.CreateWebhooksResponse{}, err
	}

	configs := make([]models.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	for eventType, w := range p.supportedWebhooks {
		configs = append(configs, models.PSPWebhookConfig{
			Name:    string(eventType),
			URLPath: w.urlPath,
			Metadata: map[string]string{
				webhookSecretMetadataKey: secretKey,
			},
		})
	}

	return models.CreateWebhooksResponse{
		Configs: configs,
	}, nil
}

func (p *Plugin) deleteWebhooks(ctx context.Context, req models.UninstallRequest) error {
	auths, err := p.client.ListWebhookAuths(ctx)
	if err != nil {
		return err
	}

	for _, auth := range auths {
		if auth.Name == p.name {
			if err := p.client.DeleteWebhookAuth(ctx, auth.ID); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (p *Plugin) verifyWebhook(_ context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	signatureDate, ok := req.Webhook.Headers["Bi-Signature-Date"]
	if !ok || len(signatureDate) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens signature date header: %w", models.ErrWebhookVerification)
	}

	signature, ok := req.Webhook.Headers["Bi-Signature"]
	if !ok || len(signature) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens signature header: %w", models.ErrWebhookVerification)
	}

	u, err := url.Parse(req.Config.FullURL)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid powens url: %w", models.ErrWebhookVerification)
	}

	secretKey, ok := req.Config.Metadata[webhookSecretMetadataKey]
	if !ok {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens secret key: %w", models.ErrWebhookVerification)
	}

	messageToSign := fmt.Sprintf("POST.%s.%s.%s", u.Path, signatureDate[0], string(req.Webhook.Body))
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(messageToSign))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	sigBytes, err := base64.StdEncoding.DecodeString(signature[0])
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid signature encoding: %w", models.ErrWebhookVerification)
	}

	expBytes, err := base64.StdEncoding.DecodeString(expectedSignature)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid signature encoding: %w", models.ErrWebhookVerification)
	}

	if !hmac.Equal(expBytes, sigBytes) {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid powens signature: %w", models.ErrWebhookVerification)
	}

	return models.VerifyWebhookResponse{}, nil
}

func (p *Plugin) handleUserDeleted(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	var webhook client.UserDeletedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	return []models.WebhookResponse{
		{
			UserDisconnected: &models.PSPUserDisconnected{
				PSPUserID: strconv.Itoa(webhook.UserID),
			},
		},
	}, nil
}

func (p *Plugin) trimConnectionSynced(_ context.Context, req models.TrimWebhookRequest) (models.TrimWebhookResponse, error) {
	if len(req.Webhook.Body) == 0 {
		return models.TrimWebhookResponse{}, fmt.Errorf("missing powens accounts synced webhook body: %w", models.ErrValidation)
	}

	var webhook client.ConnectionSyncedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return models.TrimWebhookResponse{}, err
	}

	body, err := json.Marshal(webhook)
	if err != nil {
		return models.TrimWebhookResponse{}, err
	}

	return models.TrimWebhookResponse{
		Webhooks: []models.PSPWebhook{
			{
				BasicAuth:   req.Webhook.BasicAuth,
				QueryValues: req.Webhook.QueryValues,
				Headers:     req.Webhook.Headers,
				Body:        body,
			},
		},
	}, nil
}

func (p *Plugin) handleConnectionSynced(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	var webhook client.ConnectionSyncedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	switch webhook.Connection.State {
	case "null", "":
		return []models.WebhookResponse{
			{
				UserConnectionReconnected: &models.PSPUserConnectionReconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					At:           time.Now().UTC(),
				},
			},
		}, nil
	case "SCARequired", "webauthRequired", "additionalInformationNeeded",
		"decoupled", "actionNeeded", "wrongpass", "passwordExpired":
		var reason *string
		if webhook.Connection.ErrorMessage != "" {
			reason = pointer.For(webhook.Connection.ErrorMessage)
		} else {
			reason = pointer.For("SCA error")
		}

		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					At:           time.Now().UTC(),
					ErrorType:    models.ConnectionDisconnectedErrorTypeUserActionNeeded,
					Reason:       reason,
				},
			},
		}, nil
	case "validating":
		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					At:           time.Now().UTC(),
					ErrorType:    models.ConnectionDisconnectedErrorTypeUserActionNeeded,
					Reason:       pointer.For("temporary error: validation in progress"),
				},
			},
		}, nil
	case "rateLimiting":
		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					At:           time.Now().UTC(),
					ErrorType:    models.ConnectionDisconnectedErrorTypeTemporaryError,
					Reason:       pointer.For("temporary error: rate limiting"),
				},
			},
		}, nil
	case "websiteUnavailable":
		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					ErrorType:    models.ConnectionDisconnectedErrorTypeTemporaryError,
					At:           time.Now().UTC(),
					Reason:       pointer.For("non recoverable error: website unavailable"),
				},
			},
		}, nil
	case "bug":
		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					ErrorType:    models.ConnectionDisconnectedErrorTypeNonRecoverable,
					At:           time.Now().UTC(),
					Reason:       pointer.For("powens internal error: please contact support"),
				},
			},
		}, nil
	default:
		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					ErrorType:    models.ConnectionDisconnectedErrorTypeNonRecoverable,
					At:           time.Now().UTC(),
					Reason:       pointer.For("other errors: please contact support"),
				},
			},
		}, nil
	}
}

func (p *Plugin) handleConnectionDeleted(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	var webhook client.ConnectionDeletedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	return []models.WebhookResponse{
		{
			UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
				ConnectionID: strconv.Itoa(webhook.ConnectionID),
				ErrorType:    models.ConnectionDisconnectedErrorTypeUserActionNeeded,
				At:           time.Now().UTC(),
			},
		},
	}, nil
}

func (p *Plugin) handleAccountsFetched(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	var webhook client.AccountFetchedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	resp := make([]models.WebhookResponse, 0, len(webhook.Connection.Accounts))
	for _, account := range webhook.Connection.Accounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		pspAccount := models.PSPAccount{
			Reference:    strconv.Itoa(account.ID),
			CreatedAt:    time.Now().UTC(),
			Name:         &account.OriginalName,
			DefaultAsset: pointer.For(currency.FormatAssetWithPrecision(account.Currency.ID, account.Currency.Precision)),
			Raw:          raw,
		}

		if account.Error != "" {
			pspAccount.Metadata = map[string]string{
				"error": account.Error,
			}
		}

		resp = append(resp, models.WebhookResponse{
			OpenBankingAccount: &models.PSPOpenBankingAccount{
				PSPAccount:              pspAccount,
				OpenBankingUserID:       pointer.For(strconv.Itoa(webhook.User.ID)),
				OpenBankingConnectionID: pointer.For(strconv.Itoa(webhook.Connection.ID)),
			},
		})
	}

	return resp, nil
}

func (p *Plugin) trimAccountsSynced(_ context.Context, req models.TrimWebhookRequest) (models.TrimWebhookResponse, error) {
	if len(req.Webhook.Body) == 0 {
		return models.TrimWebhookResponse{}, fmt.Errorf("missing powens accounts synced webhook body: %w", models.ErrValidation)
	}

	var webhook client.AccountSyncedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return models.TrimWebhookResponse{}, err
	}

	webhooks := make([]models.PSPWebhook, 0, len(webhook.Transactions))
	index := 0
	for {
		accountSyncedWebhook := client.AccountSyncedWebhook{
			BankAccountID: webhook.BankAccountID,
			ConnectionID:  webhook.ConnectionID,
			UserID:        webhook.UserID,
			Name:          webhook.Name,
			LastUpdate:    webhook.LastUpdate,
			Currency:      webhook.Currency,
			Transactions:  make([]client.Transaction, 0, webhookAccountSyncedTransactionsLimit),
		}

		limit := index + webhookAccountSyncedTransactionsLimit
		for ; index < len(webhook.Transactions) && index < limit; index++ {
			accountSyncedWebhook.Transactions = append(accountSyncedWebhook.Transactions, webhook.Transactions[index])
		}

		if len(accountSyncedWebhook.Transactions) > 0 {
			body, err := json.Marshal(accountSyncedWebhook)
			if err != nil {
				return models.TrimWebhookResponse{}, err
			}

			w := models.PSPWebhook{
				BasicAuth:   req.Webhook.BasicAuth,
				QueryValues: req.Webhook.QueryValues,
				Headers:     req.Webhook.Headers,
				Body:        body,
			}

			webhooks = append(webhooks, w)
		}

		if index >= len(webhook.Transactions) {
			break
		}
	}

	return models.TrimWebhookResponse{
		Webhooks: webhooks,
	}, nil
}

func (p *Plugin) handleAccountSynced(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	var webhook client.AccountSyncedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	responses := make([]models.WebhookResponse, 0, len(webhook.Transactions))
	for _, transaction := range webhook.Transactions {
		payment, err := translateTransactionToPSPPayment(transaction, webhook.Currency.ID, webhook.Currency.Precision)
		if err != nil {
			return nil, err
		}

		openBankingPayment := models.PSPOpenBankingPayment{
			PSPPayment:              payment,
			OpenBankingUserID:       pointer.For(strconv.Itoa(webhook.UserID)),
			OpenBankingConnectionID: pointer.For(strconv.Itoa(webhook.ConnectionID)),
		}

		responses = append(responses, models.WebhookResponse{
			OpenBankingPayment: &openBankingPayment,
		})
	}

	return responses, nil
}

func translateTransactionToPSPPayment(transaction client.Transaction, curr string, precision int) (models.PSPPayment, error) {
	paymentType := models.PAYMENT_TYPE_PAYIN
	if transaction.Value < 0 {
		paymentType = models.PAYMENT_TYPE_PAYOUT
	}

	amountString := strconv.FormatFloat(math.Abs(transaction.Value), 'f', -1, 64)

	amount, err := currency.GetAmountWithPrecisionFromString(amountString, precision)
	if err != nil {
		return models.PSPPayment{}, err
	}

	raw, err := json.Marshal(transaction)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		Reference: strconv.Itoa(transaction.ID),
		CreatedAt: transaction.Date,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAssetWithPrecision(curr, precision),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       raw,
	}, nil
}
