package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Address struct {
	StreetName   *string `json:"streetName"`
	StreetNumber *string `json:"streetNumber"`
	City         *string `json:"city"`
	Region       *string `json:"region"`
	PostalCode   *string `json:"postalCode"`
	Country      *string `json:"country"`
}

type ContactDetails struct {
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phoneNumber"`
	Locale      *string `json:"locale"`
}

type PSPPaymentServiceUser struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`

	// Optional fields
	ContactDetails *ContactDetails   `json:"contactDetails"`
	Address        *Address          `json:"address"`
	Metadata       map[string]string `json:"metadata"`
}

type PaymentServiceUser struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`

	// Optional fields
	ContactDetails *ContactDetails   `json:"contactDetails"`
	Address        *Address          `json:"address"`
	Metadata       map[string]string `json:"metadata"`

	BankAccountIDs        []uuid.UUID `json:"bankAccountIDs"`
	BankBridgeConnections []uuid.UUID `json:"bankBridgeConnections"`
}

func (psu PaymentServiceUser) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID             string            `json:"id"`
		Name           string            `json:"name"`
		CreatedAt      time.Time         `json:"createdAt"`
		ContactDetails *ContactDetails   `json:"contactDetails"`
		Address        *Address          `json:"address"`
		BankAccountIDs []string          `json:"bankAccountIDs"`
		Metadata       map[string]string `json:"metadata"`
	}{
		ID:             psu.ID.String(),
		Name:           psu.Name,
		CreatedAt:      psu.CreatedAt,
		ContactDetails: psu.ContactDetails,
		Address:        psu.Address,
		BankAccountIDs: func() []string {
			if len(psu.BankAccountIDs) == 0 {
				return nil
			}
			bankAccountIDs := make([]string, len(psu.BankAccountIDs))
			for i, id := range psu.BankAccountIDs {
				bankAccountIDs[i] = id.String()
			}
			return bankAccountIDs
		}(),
		Metadata: psu.Metadata,
	})
}

func (psu *PaymentServiceUser) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID             string            `json:"id"`
		Name           string            `json:"name"`
		CreatedAt      time.Time         `json:"createdAt"`
		ContactDetails *ContactDetails   `json:"contactDetails"`
		Address        *Address          `json:"address"`
		BankAccountIDs []string          `json:"bankAccountIDs"`
		Metadata       map[string]string `json:"metadata"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	psu.ID, err = uuid.Parse(aux.ID)
	if err != nil {
		return err
	}
	psu.Name = aux.Name
	psu.CreatedAt = aux.CreatedAt
	psu.ContactDetails = aux.ContactDetails
	psu.Address = aux.Address
	psu.Metadata = aux.Metadata

	if len(aux.BankAccountIDs) > 0 {
		psu.BankAccountIDs = make([]uuid.UUID, len(aux.BankAccountIDs))
		for i, id := range aux.BankAccountIDs {
			psu.BankAccountIDs[i], _ = uuid.Parse(id)
		}
	} else {
		psu.BankAccountIDs = nil
	}

	return nil
}

func ToPSPPaymentServiceUser(from *PaymentServiceUser) *PSPPaymentServiceUser {
	if from == nil {
		return nil
	}

	return &PSPPaymentServiceUser{
		ID:             from.ID,
		Name:           from.Name,
		CreatedAt:      from.CreatedAt,
		ContactDetails: from.ContactDetails,
		Address:        from.Address,
		Metadata:       from.Metadata,
	}
}
