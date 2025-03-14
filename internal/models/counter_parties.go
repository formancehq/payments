package models

import (
	"time"

	"github.com/google/uuid"
)

type PSPCounterParty struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time

	// Optional fields
	ContactDetails *ContactDetails
	Address        *Address
	BankAccount    *BankAccount

	Metadata map[string]string
}

type Address struct {
	StreetName   *string `json:"streetName"`
	StreetNumber *string `json:"streetNumber"`
	City         *string `json:"city"`
	PostalCode   *string `json:"postalCode"`
	Country      *string `json:"country"`
}

type ContactDetails struct {
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type CounterParty struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`

	// Optional fields
	ContactDetails *ContactDetails `json:"contactDetails"`
	Address        *Address        `json:"address"`
	BankAccountID  *uuid.UUID      `json:"bankAccountID"`

	Metadata map[string]string `json:"metadata"`

	RelatedAccounts []CounterPartiesRelatedAccount `json:"relatedAccounts"`
}

type counterPartyIK struct {
	ID            uuid.UUID  `json:"id"`
	LastAccountID *AccountID `json:"lastAccountID,omitempty"`
}

func (c *CounterParty) IdempotencyKey() string {
	ik := counterPartyIK{
		ID: c.ID,
	}

	if len(c.RelatedAccounts) > 0 {
		ik.LastAccountID = &c.RelatedAccounts[len(c.RelatedAccounts)-1].AccountID
	}

	return IdempotencyKey(ik)
}

func ToPSPCounterParty(c *CounterParty, ba *BankAccount) PSPCounterParty {
	return PSPCounterParty{
		ID:        c.ID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,

		ContactDetails: c.ContactDetails,
		Address:        c.Address,
		BankAccount:    ba,

		Metadata: c.Metadata,
	}
}
