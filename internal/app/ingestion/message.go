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
	ID            string                            `json:"id"`
	Reference     string                            `json:"reference"`
	Accounts      interface{}                       `json:"accounts"`
	Type          models.PaymentType                `json:"type"`
	Provider      string                            `json:"provider"`
	Status        models.PaymentStatus              `json:"status"`
	InitialAmount int64                             `json:"initialAmount"`
	Scheme        models.PaymentScheme              `json:"scheme"`
	Asset         models.PaymentAsset               `json:"asset"`
	CreatedAt     time.Time                         `json:"createdAt"`
	Raw           interface{}                       `json:"raw"`
	Amount        int64                             `json:"amount"`
	Adjustments   []paymentAdjustmentMessagePayload `json:"adjustments"`
}

type paymentAdjustmentMessagePayload struct {
	Status   models.PaymentStatus `json:"status"`
	Amount   int64                `json:"amount"`
	Date     time.Time            `json:"date"`
	Raw      interface{}          `json:"raw"`
	Absolute bool                 `json:"absolute"`
}

func NewEventSavedPayments(payments []*models.Payment) EventMessage {
	payload := make([]paymentMessagePayload, len(payments))

	for paymentIdx, payment := range payments {
		payload[paymentIdx] = paymentMessagePayload{
			ID:            payment.ID.String(),
			Reference:     payment.Reference,
			Accounts:      payment.Account,
			Type:          payment.Type,
			Status:        payment.Status,
			InitialAmount: payment.Amount,
			Scheme:        payment.Scheme,
			Asset:         payment.Asset,
			CreatedAt:     payment.CreatedAt,
			Raw:           payment.RawData,
			Amount:        payment.Amount,
		}

		if payment.Account != nil {
			payload[paymentIdx].Provider = payment.Account.Provider
		}

		for _, adjustment := range payment.Adjustments {
			payload[paymentIdx].Adjustments = append(payload[paymentIdx].Adjustments, paymentAdjustmentMessagePayload{
				Status:   adjustment.Status,
				Amount:   adjustment.Amount,
				Date:     adjustment.CreatedAt,
				Raw:      adjustment.RawData,
				Absolute: adjustment.Absolute,
			})
		}
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
