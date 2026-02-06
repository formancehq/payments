package powens

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/powens/client"
	"github.com/formancehq/payments/pkg/connector"
)

const (
	webhookSecretMetadataKey = "secret"

	webhookAccountSyncedTransactionsLimit = 100
)

type supportedWebhook struct {
	urlPath        string
	trimFunction   func(context.Context, connector.TrimWebhookRequest) (connector.TrimWebhookResponse, error)
	handleFunction func(context.Context, connector.TranslateWebhookRequest) ([]connector.WebhookResponse, error)
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
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req connector.CreateWebhooksRequest) (connector.CreateWebhooksResponse, error) {
	secretKey, err := p.client.CreateWebhookAuth(ctx, p.name)
	if err != nil {
		return connector.CreateWebhooksResponse{}, err
	}

	configs := make([]connector.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	for eventType, w := range p.supportedWebhooks {
		configs = append(configs, connector.PSPWebhookConfig{
			Name:    string(eventType),
			URLPath: w.urlPath,
			Metadata: map[string]string{
				webhookSecretMetadataKey: secretKey,
			},
		})
	}

	return connector.CreateWebhooksResponse{
		Configs: configs,
	}, nil
}

func (p *Plugin) deleteWebhooks(ctx context.Context, req connector.UninstallRequest) error {
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

func (p *Plugin) verifyWebhook(_ context.Context, req connector.VerifyWebhookRequest) (connector.VerifyWebhookResponse, error) {
	signatureDate, ok := req.Webhook.Headers["Bi-Signature-Date"]
	if !ok || len(signatureDate) != 1 {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("missing powens signature date header: %w", connector.ErrWebhookVerification)
	}

	signature, ok := req.Webhook.Headers["Bi-Signature"]
	if !ok || len(signature) != 1 {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("missing powens signature header: %w", connector.ErrWebhookVerification)
	}

	u, err := url.Parse(req.Config.FullURL)
	if err != nil {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("invalid powens url: %w", connector.ErrWebhookVerification)
	}

	secretKey, ok := req.Config.Metadata[webhookSecretMetadataKey]
	if !ok {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("missing powens secret key: %w", connector.ErrWebhookVerification)
	}

	messageToSign := fmt.Sprintf("POST.%s.%s.%s", u.Path, signatureDate[0], string(req.Webhook.Body))
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(messageToSign))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	sigBytes, err := base64.StdEncoding.DecodeString(signature[0])
	if err != nil {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("invalid signature encoding: %w", connector.ErrWebhookVerification)
	}

	expBytes, err := base64.StdEncoding.DecodeString(expectedSignature)
	if err != nil {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("invalid signature encoding: %w", connector.ErrWebhookVerification)
	}

	if !hmac.Equal(expBytes, sigBytes) {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("invalid powens signature: %w", connector.ErrWebhookVerification)
	}

	return connector.VerifyWebhookResponse{}, nil
}

func (p *Plugin) handleUserDeleted(ctx context.Context, req connector.TranslateWebhookRequest) ([]connector.WebhookResponse, error) {
	var webhook client.UserDeletedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	return []connector.WebhookResponse{
		{
			UserDisconnected: &connector.PSPUserDisconnected{
				PSPUserID: strconv.Itoa(webhook.UserID),
			},
		},
	}, nil
}

func (p *Plugin) trimConnectionSynced(_ context.Context, req connector.TrimWebhookRequest) (connector.TrimWebhookResponse, error) {
	if len(req.Webhook.Body) == 0 {
		return connector.TrimWebhookResponse{}, fmt.Errorf("missing powens accounts synced webhook body: %w", connector.ErrValidation)
	}

	var webhook client.ConnectionSyncedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return connector.TrimWebhookResponse{}, err
	}

	webhooks := make([]connector.PSPWebhook, 0)
	for _, account := range webhook.Connection.Accounts {
		acc := account
		index := 0
		for {
			connectionSyncedWebhook := client.ConnectionSyncedWebhook{
				User: webhook.User,
				Connection: client.ConnectionSyncedConnection{
					ID:           webhook.Connection.ID,
					State:        webhook.Connection.State,
					ErrorMessage: webhook.Connection.ErrorMessage,
					LastUpdate:   webhook.Connection.LastUpdate,
					Active:       webhook.Connection.Active,
				},
			}

			// Add balances here
			ba := client.BankAccount{
				ID:           acc.ID,
				ConnectionID: acc.ConnectionID,
				UserID:       acc.UserID,
				OriginalName: acc.OriginalName,
				LastUpdate:   acc.LastUpdate,
				Currency:     acc.Currency,
				Balance:      acc.Balance,
				Transactions: make([]client.Transaction, 0, webhookAccountSyncedTransactionsLimit),
			}

			limit := index + webhookAccountSyncedTransactionsLimit
			for ; index < len(acc.Transactions) && index < limit; index++ {
				ba.Transactions = append(ba.Transactions, acc.Transactions[index])
			}

			connectionSyncedWebhook.Connection.Accounts = append(connectionSyncedWebhook.Connection.Accounts, ba)

			body, err := json.Marshal(connectionSyncedWebhook)
			if err != nil {
				return connector.TrimWebhookResponse{}, err
			}

			w := connector.PSPWebhook{
				BasicAuth:   req.Webhook.BasicAuth,
				QueryValues: req.Webhook.QueryValues,
				Headers:     req.Webhook.Headers,
				Body:        body,
			}

			webhooks = append(webhooks, w)

			if index >= len(acc.Transactions) {
				break
			}
		}
	}

	return connector.TrimWebhookResponse{
		Webhooks: webhooks,
	}, nil
}

func (p *Plugin) handleConnectionSynced(ctx context.Context, req connector.TranslateWebhookRequest) ([]connector.WebhookResponse, error) {
	var webhook client.ConnectionSyncedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	switch webhook.Connection.State {
	case "null", "":
		// There is only one account in the webhook since we trimmed it before
		account := webhook.Connection.Accounts[0]

		pspAccount, err := translateBankAccountToPSPAccount(account)
		if err != nil {
			return nil, err
		}

		accountsResponse := connector.WebhookResponse{
			OpenBankingAccount: &connector.PSPOpenBankingAccount{
				PSPAccount:              pspAccount,
				OpenBankingUserID:       pointer.For(strconv.Itoa(webhook.User.ID)),
				OpenBankingConnectionID: pointer.For(strconv.Itoa(webhook.Connection.ID)),
			},
		}

		transactionResponses := make([]connector.WebhookResponse, 0, len(account.Transactions))
		for _, transaction := range account.Transactions {
			payment, err := translateTransactionToPSPPayment(transaction, pspAccount.Reference, account.Currency.ID, account.Currency.Precision)
			if err != nil {
				return nil, err
			}

			obPayment := connector.PSPOpenBankingPayment{
				PSPPayment:              payment,
				OpenBankingUserID:       pointer.For(strconv.Itoa(account.UserID)),
				OpenBankingConnectionID: pointer.For(strconv.Itoa(account.ConnectionID)),
			}

			transactionResponses = append(transactionResponses, connector.WebhookResponse{
				OpenBankingPayment: &obPayment,
			})
		}

		at := time.Now().UTC()
		if !account.LastUpdate.IsZero() {
			at = account.LastUpdate
		}

		var balanceResponse *connector.WebhookResponse
		if account.Balance.String() != "" {
			amount, err := currency.GetAmountWithPrecisionFromString(account.Balance.String(), account.Currency.Precision)
			if err != nil {
				return nil, err
			}
			balanceResponse = &connector.WebhookResponse{
				Balance: &connector.PSPBalance{
					AccountReference: pspAccount.Reference,
					CreatedAt:        at,
					Asset:            *pspAccount.DefaultAsset,
					Amount:           amount,
				},
			}
		}

		// We have to put first the user connection reconnected webhook in order
		// to be sure that the connection is created before the payments and
		// accounts are ingested.
		res := []connector.WebhookResponse{
			{
				UserConnectionReconnected: &connector.PSPUserConnectionReconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					At:           at,
				},
			},
		}

		// Then create the accounts
		res = append(res, accountsResponse)
		// And finally the transactions related to the connection and the account
		res = append(res, transactionResponses...)
		// Finally, push the latest balance for the account
		if balanceResponse != nil {
			res = append(res, *balanceResponse)
		}

		return res, nil

	case "SCARequired", "webauthRequired", "additionalInformationNeeded",
		"decoupled", "actionNeeded", "wrongpass", "passwordExpired":
		var reason *string
		if webhook.Connection.ErrorMessage != "" {
			reason = pointer.For(webhook.Connection.ErrorMessage)
		} else {
			reason = pointer.For("SCA error")
		}

		return []connector.WebhookResponse{
			{
				UserConnectionDisconnected: &connector.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					At:           time.Now().UTC(),
					ErrorType:    connector.ConnectionDisconnectedErrorTypeUserActionNeeded,
					Reason:       reason,
				},
			},
		}, nil
	case "validating":
		return []connector.WebhookResponse{
			{
				UserConnectionDisconnected: &connector.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					At:           time.Now().UTC(),
					ErrorType:    connector.ConnectionDisconnectedErrorTypeUserActionNeeded,
					Reason:       pointer.For("temporary error: validation in progress"),
				},
			},
		}, nil
	case "rateLimiting":
		return []connector.WebhookResponse{
			{
				UserConnectionDisconnected: &connector.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					At:           time.Now().UTC(),
					ErrorType:    connector.ConnectionDisconnectedErrorTypeTemporaryError,
					Reason:       pointer.For("temporary error: rate limiting"),
				},
			},
		}, nil
	case "websiteUnavailable":
		return []connector.WebhookResponse{
			{
				UserConnectionDisconnected: &connector.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					ErrorType:    connector.ConnectionDisconnectedErrorTypeTemporaryError,
					At:           time.Now().UTC(),
					Reason:       pointer.For("non recoverable error: website unavailable"),
				},
			},
		}, nil
	case "bug":
		return []connector.WebhookResponse{
			{
				UserConnectionDisconnected: &connector.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					ErrorType:    connector.ConnectionDisconnectedErrorTypeNonRecoverable,
					At:           time.Now().UTC(),
					Reason:       pointer.For("powens internal error: please contact support"),
				},
			},
		}, nil
	default:
		return []connector.WebhookResponse{
			{
				UserConnectionDisconnected: &connector.PSPUserConnectionDisconnected{
					PSPUserID:    strconv.Itoa(webhook.User.ID),
					ConnectionID: strconv.Itoa(webhook.Connection.ID),
					ErrorType:    connector.ConnectionDisconnectedErrorTypeNonRecoverable,
					At:           time.Now().UTC(),
					Reason:       pointer.For("other errors: please contact support"),
				},
			},
		}, nil
	}
}

func (p *Plugin) handleConnectionDeleted(ctx context.Context, req connector.TranslateWebhookRequest) ([]connector.WebhookResponse, error) {
	var webhook client.ConnectionDeletedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	return []connector.WebhookResponse{
		{
			UserConnectionDisconnected: &connector.PSPUserConnectionDisconnected{
				ConnectionID: strconv.Itoa(webhook.ConnectionID),
				ErrorType:    connector.ConnectionDisconnectedErrorTypeUserActionNeeded,
				At:           time.Now().UTC(),
			},
		},
	}, nil
}

func translateBankAccountToPSPAccount(account client.BankAccount) (connector.PSPAccount, error) {
	acc := account
	// We don't need the transactions in the raw payload of the account
	acc.Transactions = nil
	raw, err := json.Marshal(acc)
	if err != nil {
		return connector.PSPAccount{}, err
	}

	res := connector.PSPAccount{
		Reference:    strconv.Itoa(account.ID),
		CreatedAt:    time.Now().UTC(),
		Name:         &account.OriginalName,
		DefaultAsset: pointer.For(currency.FormatAssetWithPrecision(account.Currency.ID, account.Currency.Precision)),
		Raw:          raw,
	}

	if account.Error != "" {
		res.Metadata = map[string]string{
			"error": account.Error,
		}
	}

	return res, nil
}

func translateTransactionToPSPPayment(transaction client.Transaction, accountReference string, curr string, precision int) (connector.PSPPayment, error) {

	amount, err := currency.GetAmountWithPrecisionFromString(transaction.Value.String(), precision)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	paymentType := connector.PAYMENT_TYPE_PAYIN
	if amount.Sign() == -1 {
		paymentType = connector.PAYMENT_TYPE_PAYOUT
	}

	raw, err := json.Marshal(transaction)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	p := connector.PSPPayment{
		Reference: strconv.Itoa(transaction.ID),
		CreatedAt: transaction.Date,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAssetWithPrecision(curr, precision),
		Scheme:    connector.PAYMENT_SCHEME_OTHER,
		Status:    connector.PAYMENT_STATUS_SUCCEEDED,
		Raw:       raw,
	}

	switch paymentType {
	case connector.PAYMENT_TYPE_PAYIN:
		p.DestinationAccountReference = &accountReference
	case connector.PAYMENT_TYPE_PAYOUT:
		p.SourceAccountReference = &accountReference
	}

	return p, nil
}
