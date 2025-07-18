package plaid

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/plaid/plaid-go/v34/plaid"
)

type supportedWebhook struct {
	urlPath string
	fn      func(context.Context, models.TranslateWebhookRequest) ([]models.WebhookResponse, error)
}

func (p *Plugin) initWebhookConfig() {
	p.supportedWebhooks = map[string]supportedWebhook{
		"all": {
			urlPath: "/all",
			fn:      p.handleAllWebhook,
		},
	}
}

func (p *Plugin) createWebhooks(_ context.Context, _ models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	configs := make([]models.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	for name, w := range p.supportedWebhooks {
		configs = append(configs, models.PSPWebhookConfig{
			Name:    name,
			URLPath: w.urlPath,
		})
	}

	return models.CreateWebhooksResponse{
		Configs: configs,
	}, nil
}

func (p *Plugin) verifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	// Extract the signed JWT from the webhook header
	tokenStrings := req.Webhook.Headers["Plaid-Verification"]
	if len(tokenStrings) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid token: %w", models.ErrInvalidRequest)
	}

	// Verify the token using the public key
	token, err := jwt.Parse(tokenStrings[0], func(token *jwt.Token) (interface{}, error) {
		// Extract the key ID (kid) from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header: %w", models.ErrInvalidRequest)
		}

		key, err := p.client.GetWebhookVerificationKey(ctx, kid)
		if err != nil {
			return nil, err
		}

		if key.ExpiredAt.IsSet() {
			// reject expired keys
			return nil, fmt.Errorf("expired key: %w", models.ErrInvalidRequest)
		}

		// Signing key must be an ecdsa.PublicKey struct
		publicKey := new(ecdsa.PublicKey)
		publicKey.Curve = elliptic.P256()
		x, _ := base64.URLEncoding.DecodeString(key.X + "=")
		xc := new(big.Int)
		publicKey.X = xc.SetBytes(x)
		y, _ := base64.URLEncoding.DecodeString(key.Y + "=")
		yc := new(big.Int)
		publicKey.Y = yc.SetBytes(y)

		return publicKey, nil
	}, jwt.WithValidMethods([]string{"ES256"}))
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("failed to verify token: %w", err)
	}

	// Check if the token is valid
	if !token.Valid {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid token: %w", models.ErrInvalidRequest)
	}

	return models.VerifyWebhookResponse{}, nil
}

func (p *Plugin) translateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	webhookTranslator, ok := p.supportedWebhooks[req.Name]
	if !ok {
		return models.TranslateWebhookResponse{}, fmt.Errorf("unsupported webhook event type: %s", req.Name)
	}

	resp, err := webhookTranslator.fn(ctx, req)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	return models.TranslateWebhookResponse{
		Responses: resp,
	}, nil
}

func (p *Plugin) handleAllWebhook(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	baseWebhook, err := p.client.BaseWebhookTranslation(req.Webhook.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println("handleAllWebhook", baseWebhook)

	switch baseWebhook.WebhookType {
	// This one has no type since it's not inside the plaid sdk definition
	// but we need it in order to handle the authentication webhook
	case "LINK":
		return p.handleLinkWebhook(ctx, req, baseWebhook)
	case plaid.WEBHOOKTYPE_ASSETS,
		plaid.WEBHOOKTYPE_AUTH,
		plaid.WEBHOOKTYPE_HOLDINGS,
		plaid.WEBHOOKTYPE_INVESTMENTS_TRANSACTIONS,
		plaid.WEBHOOKTYPE_LIABILITIES:
		// Nothing to do for these webhooks
		return []models.WebhookResponse{}, nil

	case plaid.WEBHOOKTYPE_ITEM:
		return p.handleItemWebhook(req, baseWebhook)

	case plaid.WEBHOOKTYPE_TRANSACTIONS:
		return p.handleTransactionsWebhook(req, baseWebhook)

	default:
		return []models.WebhookResponse{}, fmt.Errorf("unsupported webhook type: %s", baseWebhook.WebhookType)
	}
}

func (p *Plugin) handleLinkWebhook(ctx context.Context, req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	switch baseWebhook.WebhookCode {
	case "ITEM_ADD_RESULT":
		return []models.WebhookResponse{}, nil
	case "SESSION_FINISHED":
		fmt.Println("handleSessionFinishedWebhook", string(req.Webhook.Body))
		return p.handleSessionFinishedWebhook(ctx, req)
	case "EVENTS":
		// Note: Nothing to do for us here for now.
		return []models.WebhookResponse{}, nil
	}

	return []models.WebhookResponse{}, nil
}

func (p *Plugin) handleSessionFinishedWebhook(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	webhook, err := p.client.TranslateSessionFinishedWebhook(req.Webhook.Body)
	if err != nil {
		return nil, err
	}

	ids, ok := req.Webhook.QueryValues[client.AttemptIDQueryParamID]
	if !ok || len(ids) != 1 {
		return nil, fmt.Errorf("missing attemptID: %w", models.ErrInvalidRequest)
	}

	attemptID, err := uuid.Parse(ids[0])
	if err != nil {
		return nil, fmt.Errorf("invalid attemptID: %w", models.ErrInvalidRequest)
	}

	status := models.PSUBankBridgeConnectionAttemptStatusPending
	var errMsg *string
	switch strings.ToLower(webhook.GetStatus()) {
	case "success":
		status = models.PSUBankBridgeConnectionAttemptStatusCompleted
	case "exited":
		errMsg = pointer.For("exited")
		status = models.PSUBankBridgeConnectionAttemptStatusFailed
	}

	for _, publicToken := range *webhook.PublicTokens {
		if err := p.client.FormanceBankBridgeRedirect(ctx, client.FormanceBankBridgeRedirectRequest{
			LinkToken:   webhook.LinkToken,
			PublicToken: publicToken,
			AttemptID:   attemptID,
		}); err != nil {
			return nil, err
		}
	}

	fmt.Println("handleSessionFinishedWebhook", status, errMsg)
	return []models.WebhookResponse{
		{
			UserLinkSessionFinished: &models.PSPUserLinkSessionFinished{
				AttemptID: attemptID,
				Status:    status,
				Error:     errMsg,
			},
		},
	}, nil
}

func (p *Plugin) handleItemWebhook(req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	switch baseWebhook.WebhookCode {
	case "ERROR":
		// Fired when an error is encountered with an Item. The error can be
		// resolved by having the user go through Link’s update mode.

		// Note: Launch update mode flow.

		webhook, err := p.client.TranslateItemErrorWebhook(req.Webhook.Body)
		if err != nil {
			return nil, err
		}

		switch webhook.Error.Get().GetErrorCode() {
		case "ITEM_LOGIN_REQUIRED":
			// Note: Launch update mode flow.
			return []models.WebhookResponse{
				{
					UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
						ConnectionID: baseWebhook.ItemID,
						At:           time.Now().UTC(),
						Reason:       pointer.For(webhook.GetError().ErrorMessage),
					},
				},
			}, nil
		}

	case "LOGIN_REPAIRED":
		fmt.Println("login repaired", string(req.Webhook.Body))
		// Fired when an Item has exited the ITEM_LOGIN_REQUIRED state without
		// the user having gone through the update mode flow in your app (this
		// can happen if the user completed the update mode in a different app).
		// If you have messaging that tells the user to complete the update mode
		// flow, you should silence this messaging upon receiving the
		// LOGIN_REPAIRED webhook.

		// Note: Nothing to do for us here for now.

	case "NEW_ACCOUNTS_AVAILABLE":
		// Fired when Plaid detects a new account. Upon receiving this webhook,
		// you can prompt your users to share new accounts with you through
		// update mode (US/CA only). If the end user has opted not to share new
		// accounts with Plaid via their institution's OAuth settings, Plaid
		// will not detect new accounts and this webhook will not fire. For end
		// user accounts in the EU and UK, upon receiving this webhook, you can
		// prompt your user to re-link their account and then delete the old
		// Item via /item/remove.

		// Note: We don't need to do anything here for now, maybe we can
		// use this to trigger a sync of the new accounts though the update mode
		// in future updates of this connector.

	case "PENDING_DISCONNECT":
		// Fired when an Item is expected to be disconnected. The webhook will
		// currently be fired 7 days before the existing Item is scheduled for
		// disconnection. This can be resolved by having the user go through
		// Link’s update mode. Currently, this webhook is fired only for US or
		// Canadian institutions; in the UK or EU, you should continue to listed
		// for the PENDING_EXPIRATION webhook instead.

		// Note: Launch update mode flow.

		webhook, err := p.client.TranslateUserPendingDisconnectWebhook(req.Webhook.Body)
		if err != nil {
			return nil, err
		}

		return []models.WebhookResponse{
			{
				UserConnectionPendingDisconnect: &models.PSPUserConnectionPendingDisconnect{
					ConnectionID: webhook.ItemId,
					Reason:       pointer.For(string(webhook.Reason)),
					At:           time.Now().UTC().Add(7 * 24 * time.Hour),
				},
			},
		}, nil

	case "PENDING_EXPIRATION":
		// Fired when an Item’s access consent is expiring in 7 days. This can
		// be resolved by having the user go through Link’s update mode. This
		// webhook is fired only for Items associated with institutions in
		// Europe (including the UK); for Items associated with institutions in
		// the US or Canada, see PENDING_DISCONNECT instead.

		// Note: Launch update mode flow.

		webhook, err := p.client.TranslateUserPendingExpirationWebhook(req.Webhook.Body)
		if err != nil {
			return nil, err
		}

		return []models.WebhookResponse{
			{
				UserConnectionPendingDisconnect: &models.PSPUserConnectionPendingDisconnect{
					ConnectionID: webhook.ItemId,
					At:           webhook.ConsentExpirationTime,
				},
			},
		}, nil

	case "USER_PERMISSION_REVOKED":
		// The USER_PERMISSION_REVOKED webhook may be fired when an end user has
		// revoked the permission that they previously granted to access an
		// Item. If the end user revoked their permissions through Plaid (such
		// as via the Plaid Portal or by contacting Plaid Support), the webhook
		// will fire. If the end user revoked their permissions directly through
		// the institution, this webhook may not always fire, since some
		// institutions’ consent portals do not trigger this webhook. Upon
		// receiving this webhook, it is recommended to delete any stored data
		// from Plaid associated with the Item. To restore the Item, it can be
		// sent through update mode.

		// Note: Delete all data associated with the Item.

		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					ConnectionID: baseWebhook.ItemID,
					At:           time.Now().UTC(),
				},
			},
		}, nil

	case "USER_ACCOUNT_REVOKED":
		// The USER_ACCOUNT_REVOKED webhook is fired when an end user has
		// revoked access to their account on the Data Provider's portal. This
		// webhook is currently sent only for Chase and PNC Items, but may be
		// sent in the future for other financial institutions that allow
		// account-level permissions revocation through their portals. Upon
		// receiving this webhook, it is recommended to delete any Plaid-derived
		// data you have stored that is associated with the revoked account. You
		// can request the user to re-grant access to their account by sending
		// them through update mode. Alternatively, they may re-grant access
		// directly through the Data Provider's portal.

		// Note: Delete all data associated with the Item.

		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					ConnectionID: baseWebhook.ItemID,
					At:           time.Now().UTC(),
				},
			},
		}, nil

	case "WEBHOOK_UPDATE_ACKNOWLEDGED":
		// Fired when an Item's webhook is updated. This will be sent to the
		// newly specified webhook.

		// Note: Nothing to do for us here for now.
	}

	return []models.WebhookResponse{}, nil
}

func (p *Plugin) handleTransactionsWebhook(req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	switch baseWebhook.WebhookCode {
	case "SYNC_UPDATES_AVAILABLE":
		// Fired when an Item's transactions change. This can be due to any
		// event resulting in new changes, such as an initial 30-day
		// transactions fetch upon the initialization of an Item with
		// transactions, the backfill of historical transactions that occurs
		// shortly after, or when changes are populated from a
		// regularly-scheduled transactions update job. It is recommended to
		// listen for the SYNC_UPDATES_AVAILABLE webhook when using the
		// /transactions/sync endpoint. Note that when using /transactions/sync
		// the older webhooks INITIAL_UPDATE, HISTORICAL_UPDATE, DEFAULT_UPDATE,
		// and TRANSACTIONS_REMOVED, which are intended for use with
		// /transactions/get, will also continue to be sent in order to maintain
		// backwards compatibility. It is not necessary to listen for and
		// respond to those webhooks when using /transactions/sync.

		// After receipt of this webhook, the new changes can be fetched for the
		// Item from /transactions/sync. Note that to receive this webhook for
		// an Item, /transactions/sync must have been called at least once on
		// that Item. This means that, unlike the INITIAL_UPDATE and
		// HISTORICAL_UPDATE webhooks, it will not fire immediately upon Item
		// creation. If /transactions/sync is called on an Item that was not
		// initialized with Transactions, the webhook will fire twice: once the
		// first 30 days of transactions data has been fetched, and a second
		// time when all available historical transactions data has been fetched.

		// Note: launch a sync of the transactions for the Item

		return []models.WebhookResponse{
			{
				DataReadyToFetch: &models.PSPDataReadyToFetch{
					ID:          &baseWebhook.ItemID,
					FromPayload: req.Webhook.Body,
				},
			},
		}, nil

	case "HISTORICAL_UPDATE":
		// In our case, we want to wait for this webhook before calling the
		// transactions sync flow.

		return []models.WebhookResponse{
			{
				DataReadyToFetch: &models.PSPDataReadyToFetch{
					ID:          &baseWebhook.ItemID,
					FromPayload: req.Webhook.Body,
				},
			},
		}, nil

	case "RECURRING_TRANSACTIONS_UPDATE":
		// Fired when recurring transactions data is updated. This includes when
		// a new recurring stream is detected or when a new transaction is added
		// to an existing recurring stream. The RECURRING_TRANSACTIONS_UPDATE
		// webhook will also fire when one or more attributes of the recurring
		// stream changes, which is usually a result of the addition, update, or
		// removal of transactions to the stream.

		// Note: We don't need to do anything here for now.

		return []models.WebhookResponse{}, nil

	case "INITIAL_UPDATE",
		"DEFAULT_UPDATE",
		"TRANSACTIONS_REMOVED":
		// Note: as specified in the docs (and also in the comment above), we
		// don't need to do anything here for now as they are deprecated
		// webhooks.

		return []models.WebhookResponse{}, nil

	default:
		return []models.WebhookResponse{}, nil
	}
}
