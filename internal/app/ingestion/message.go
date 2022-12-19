package ingestion

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/app/models"

	"github.com/formancehq/go-libs/sharedlogging"
)

const (
	TopicPayments = "payments"
	TopicAccounts = "payments"

	EventVersion = "v1"
	EventApp     = "payments"

	EventTypeSavedPayments = "SAVED_PAYMENT"
	EventTypeSavedAccounts = "SAVED_ACCOUNT"
)

type EventMessage struct {
	Date    time.Time `json:"date"`
	App     string    `json:"app"`
	Version string    `json:"version"`
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
}

type paymentMessagePayload struct {
	ID        string               `json:"id"`
	Reference string               `json:"reference"`
	CreatedAt time.Time            `json:"createdAt"`
	Provider  string               `json:"provider"`
	Type      models.PaymentType   `json:"type"`
	Status    models.PaymentStatus `json:"status"`
	Scheme    models.PaymentScheme `json:"scheme"`
	Asset     models.PaymentAsset  `json:"asset"`

	// TODO: Remove 'initialAmount' once frontend has switched to 'amount
	InitialAmount int64 `json:"initialAmount"`
	Amount        int64 `json:"amount"`
}

func NewEventSavedPayments(payment *models.Payment, provider models.ConnectorProvider) EventMessage {
	payload := paymentMessagePayload{
		ID:            payment.ID.String(),
		Reference:     payment.Reference,
		Type:          payment.Type,
		Status:        payment.Status,
		InitialAmount: payment.Amount,
		Scheme:        payment.Scheme,
		Asset:         payment.Asset,
		CreatedAt:     payment.CreatedAt,
		Amount:        payment.Amount,
		Provider:      provider.String(),
	}

	return EventMessage{
		Date:    time.Now().UTC(),
		App:     EventApp,
		Version: EventVersion,
		Type:    EventTypeSavedPayments,
		Payload: payload,
	}
}

type accountMessagePayload struct {
	ID        string             `json:"id"`
	CreatedAt time.Time          `json:"createdAt"`
	Reference string             `json:"reference"`
	Provider  string             `json:"provider"`
	Type      models.AccountType `json:"type"`
}

func NewEventSavedAccounts(accounts []models.Account) EventMessage {
	payload := make([]accountMessagePayload, len(accounts))

	for accountIdx, account := range accounts {
		payload[accountIdx] = accountMessagePayload{
			ID:        account.ID.String(),
			CreatedAt: account.CreatedAt,
			Reference: account.Reference,
			Provider:  account.Provider,
			Type:      account.Type,
		}
	}

	return EventMessage{
		Date:    time.Now().UTC(),
		App:     EventApp,
		Version: EventVersion,
		Type:    EventTypeSavedAccounts,
		Payload: payload,
	}
}

func (i *DefaultIngester) publish(ctx context.Context, topic string, ev EventMessage) {
	if err := i.publisher.Publish(ctx, topic, ev); err != nil {
		sharedlogging.GetLogger(ctx).Errorf("Publishing message: %s", err)

		return
	}
}
