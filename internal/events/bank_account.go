package events

import (
	"time"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type BankAccountMessagePayload struct {
	// Mandatory fields
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`

	// Optional fields
	AccountNumber   string                              `json:"accountNumber,omitempty"`
	IBAN            string                              `json:"iban,omitempty"`
	SwiftBicCode    string                              `json:"swiftBicCode,omitempty"`
	Country         string                              `json:"country,omitempty"`
	Metadata        map[string]string                   `json:"metadata,omitempty"`
	RelatedAccounts []BankAccountRelatedAccountsPayload `json:"relatedAccounts,omitempty"`
}

type BankAccountRelatedAccountsPayload struct {
	CreatedAt   time.Time `json:"createdAt"`
	AccountID   string    `json:"accountID"`
	ConnectorID string    `json:"connectorID"`
	Provider    string    `json:"provider"`
}

func (e Events) NewEventSavedBankAccounts(bankAccount models.BankAccount) (publish.EventMessage, error) {
	if err := bankAccount.Obfuscate(); err != nil {
		return publish.EventMessage{}, err
	}

	payload := BankAccountMessagePayload{
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
		relatedAccount := BankAccountRelatedAccountsPayload{
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
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedBankAccount,
		Payload:        payload,
	}, nil
}
