package events

import (
	"time"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type V3BankAccountMessagePayload struct {
	// Mandatory fields
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`

	// Optional fields
	AccountNumber   string                                `json:"accountNumber,omitempty"`
	IBAN            string                                `json:"iban,omitempty"`
	SwiftBicCode    string                                `json:"swiftBicCode,omitempty"`
	Country         string                                `json:"country,omitempty"`
	Metadata        map[string]string                     `json:"metadata,omitempty"`
	RelatedAccounts []V3BankAccountRelatedAccountsPayload `json:"relatedAccounts,omitempty"`
}

type V3BankAccountRelatedAccountsPayload struct {
	CreatedAt   time.Time `json:"createdAt"`
	AccountID   string    `json:"accountID"`
	ConnectorID string    `json:"connectorID"`
	Provider    string    `json:"provider"`
}

type V2BankAccountMessagePayload struct {
	ID              string                                `json:"id"`
	CreatedAt       time.Time                             `json:"createdAt"`
	Name            string                                `json:"name"`
	AccountNumber   string                                `json:"accountNumber"`
	IBAN            string                                `json:"iban"`
	SwiftBicCode    string                                `json:"swiftBicCode"`
	Country         string                                `json:"country"`
	RelatedAccounts []V2BankAccountRelatedAccountsPayload `json:"adjustments"`
}

type V2BankAccountRelatedAccountsPayload struct {
	CreatedAt   time.Time `json:"createdAt"`
	AccountID   string    `json:"accountID"`
	ConnectorID string    `json:"connectorID"`
	Provider    string    `json:"provider"`
}

func (e Events) NewEventSavedBankAccounts(bankAccount models.BankAccount) ([]publish.EventMessage, error) {
	if err := bankAccount.Obfuscate(); err != nil {
		return nil, err
	}

	return []publish.EventMessage{
		toV2BankAccountEvent(bankAccount),
		toV3BankAccountEvent(bankAccount),
	}, nil
}

func toV3BankAccountEvent(bankAccount models.BankAccount) publish.EventMessage {
	payload := V3BankAccountMessagePayload{
		ID:        bankAccount.ID.String(),
		CreatedAt: bankAccount.CreatedAt,
		Name:      bankAccount.Name,
		Metadata:  bankAccount.Metadata,
	}

	if bankAccount.AccountNumber != nil {
		payload.AccountNumber = *bankAccount.AccountNumber
	}

	if bankAccount.IBAN != nil {
		payload.IBAN = *bankAccount.IBAN
	}

	if bankAccount.SwiftBicCode != nil {
		payload.SwiftBicCode = *bankAccount.SwiftBicCode
	}

	if bankAccount.Country != nil {
		payload.Country = *bankAccount.Country
	}

	for _, relatedAccount := range bankAccount.RelatedAccounts {
		relatedAccount := V3BankAccountRelatedAccountsPayload{
			CreatedAt:   relatedAccount.CreatedAt,
			AccountID:   relatedAccount.AccountID.String(),
			Provider:    models.ToV3Provider(relatedAccount.AccountID.ConnectorID.Provider),
			ConnectorID: relatedAccount.AccountID.ConnectorID.String(),
		}

		payload.RelatedAccounts = append(payload.RelatedAccounts, relatedAccount)
	}

	return publish.EventMessage{
		IdempotencyKey: bankAccount.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.V3EventVersion,
		Type:           events.V3EventTypeSavedBankAccount,
		Payload:        payload,
	}
}

func toV2BankAccountEvent(bankAccount models.BankAccount) publish.EventMessage {
	payload := V2BankAccountMessagePayload{
		ID:        bankAccount.ID.String(),
		CreatedAt: bankAccount.CreatedAt,
		Name:      bankAccount.Name,
	}

	if bankAccount.AccountNumber != nil {
		payload.AccountNumber = *bankAccount.AccountNumber
	}

	if bankAccount.IBAN != nil {
		payload.IBAN = *bankAccount.IBAN
	}

	if bankAccount.SwiftBicCode != nil {
		payload.SwiftBicCode = *bankAccount.SwiftBicCode
	}

	if bankAccount.Country != nil {
		payload.Country = *bankAccount.Country
	}

	for _, relatedAccount := range bankAccount.RelatedAccounts {
		relatedAccount := V2BankAccountRelatedAccountsPayload{
			CreatedAt:   relatedAccount.CreatedAt,
			AccountID:   relatedAccount.AccountID.String(),
			Provider:    models.ToV2Provider(relatedAccount.AccountID.ConnectorID.Provider),
			ConnectorID: relatedAccount.AccountID.ConnectorID.String(),
		}

		payload.RelatedAccounts = append(payload.RelatedAccounts, relatedAccount)
	}

	return publish.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.V2EventVersion,
		Type:    events.V2EventTypeSavedBankAccount,
		Payload: payload,
	}
}
