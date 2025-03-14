package events

import (
	"time"

	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type CounterPartyAddress struct {
	StreetName   *string `json:"streetName,omitempty"`
	StreetNumber *string `json:"streetNumber,omitempty"`
	City         *string `json:"city,omitempty"`
	PostalCode   *string `json:"postalCode,omitempty"`
	Country      *string `json:"country,omitempty"`
}

type CounterPartyContactDetails struct {
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

type CounterPartyMessagePayload struct {
	ID              string                               `json:"id"`
	CreatedAt       time.Time                            `json:"createdAt"`
	Name            string                               `json:"name"`
	ContactDetails  *CounterPartyContactDetails          `json:"contactDetails,omitempty"`
	Address         *CounterPartyAddress                 `json:"address,omitempty"`
	BankAccountID   *string                              `json:"bankAccountID,omitempty"`
	RelatedAccounts []CounterPartyRelatedAccountsPayload `json:"relatedAccounts"`
}

type CounterPartyRelatedAccountsPayload struct {
	CreatedAt   time.Time `json:"createdAt"`
	AccountID   string    `json:"accountID"`
	ConnectorID string    `json:"connectorID"`
	Provider    string    `json:"provider"`
}

func (e Events) NewEventSavedCounterParty(counterParty models.CounterParty) publish.EventMessage {
	payload := CounterPartyMessagePayload{
		ID:        counterParty.ID.String(),
		CreatedAt: counterParty.CreatedAt,
		Name:      counterParty.Name,
	}

	if counterParty.ContactDetails != nil {
		payload.ContactDetails = &CounterPartyContactDetails{
			Email: counterParty.ContactDetails.Email,
			Phone: counterParty.ContactDetails.Phone,
		}
	}

	if counterParty.Address != nil {
		payload.Address = &CounterPartyAddress{
			StreetName:   counterParty.Address.StreetName,
			StreetNumber: counterParty.Address.StreetNumber,
			City:         counterParty.Address.City,
			PostalCode:   counterParty.Address.PostalCode,
			Country:      counterParty.Address.Country,
		}
	}

	if counterParty.BankAccountID != nil {
		payload.BankAccountID = pointer.For(counterParty.BankAccountID.String())
	}

	for _, relatedAccount := range counterParty.RelatedAccounts {
		relatedAccount := CounterPartyRelatedAccountsPayload{
			CreatedAt:   relatedAccount.CreatedAt,
			AccountID:   relatedAccount.AccountID.String(),
			Provider:    relatedAccount.AccountID.ConnectorID.Provider,
			ConnectorID: relatedAccount.AccountID.ConnectorID.String(),
		}

		payload.RelatedAccounts = append(payload.RelatedAccounts, relatedAccount)
	}

	return publish.EventMessage{
		IdempotencyKey: counterParty.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedCounterParty,
		Payload:        payload,
	}
}
