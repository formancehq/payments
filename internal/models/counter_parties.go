package models

import (
	"time"

	"github.com/google/uuid"
)

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

func (c *CounterParty) IdempotencyKey() string {
	return IdempotencyKey(c.ID)
}
