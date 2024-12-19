package models

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
)

type PSPPaymentInitiation struct {
	// Reference of the unique payment initiation
	Reference string

	// Payment Initiation creation date
	CreatedAt time.Time

	// Description of the payment
	Description string

	// PSP reference of the source account
	SourceAccount *PSPAccount
	// PSP reference of the destination account
	DestinationAccount *PSPAccount

	// Amount of the payment
	Amount *big.Int
	// Asset of the payment
	Asset string

	// Additional metadata
	Metadata map[string]string
}

type PaymentInitiation struct {
	// Unique Payment initiation ID generated from payments information
	ID PaymentInitiationID `json:"id"`
	// Related Connector ID
	ConnectorID ConnectorID `json:"connectorID"`
	// Unique reference of the payment
	Reference string `json:"reference"`

	// Payment Initiation creation date
	CreatedAt time.Time `json:"createdAt"`

	// Time to schedule the payment
	ScheduledAt time.Time `json:"scheduledAt"`

	// Description of the payment
	Description string `json:"description"`

	Type PaymentInitiationType `json:"paymentInitiationType"`

	// Source account of the payment
	SourceAccountID *AccountID `json:"sourceAccountID"`
	// Destination account of the payment
	DestinationAccountID *AccountID `json:"destinationAccountID"`

	// Payment current amount (can be changed in case of reversed, refunded, etc...)
	Amount *big.Int `json:"amount"`
	// Asset of the payment
	Asset string `json:"asset"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`
}

func (pi PaymentInitiation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                   string                `json:"id"`
		ConnectorID          string                `json:"connectorID"`
		Reference            string                `json:"reference"`
		CreatedAt            time.Time             `json:"createdAt"`
		ScheduledAt          time.Time             `json:"scheduledAt"`
		Description          string                `json:"description"`
		Type                 PaymentInitiationType `json:"paymentInitiationType"`
		SourceAccountID      *string               `json:"sourceAccountID,omitempty"`
		DestinationAccountID *string               `json:"destinationAccountID,omitempty"`
		Amount               *big.Int              `json:"amount"`
		Asset                string                `json:"asset"`
		Metadata             map[string]string     `json:"metadata"`
	}{
		ID:          pi.ID.String(),
		ConnectorID: pi.ConnectorID.String(),
		Reference:   pi.Reference,
		CreatedAt:   pi.CreatedAt,
		ScheduledAt: pi.ScheduledAt,
		Description: pi.Description,
		Type:        pi.Type,
		SourceAccountID: func() *string {
			if pi.SourceAccountID == nil {
				return nil
			}
			return pointer.For(pi.SourceAccountID.String())
		}(),
		DestinationAccountID: func() *string {
			if pi.DestinationAccountID == nil {
				return nil
			}
			return pointer.For(pi.DestinationAccountID.String())
		}(),
		Amount:   pi.Amount,
		Asset:    pi.Asset,
		Metadata: pi.Metadata,
	})
}

func (pi *PaymentInitiation) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                   string                `json:"id"`
		ConnectorID          string                `json:"connectorID"`
		Reference            string                `json:"reference"`
		CreatedAt            time.Time             `json:"createdAt"`
		ScheduledAt          time.Time             `json:"scheduledAt"`
		Description          string                `json:"description"`
		Type                 PaymentInitiationType `json:"paymentInitiationType"`
		SourceAccountID      *string               `json:"sourceAccountID,omitempty"`
		DestinationAccountID *string               `json:"destinationAccountID,omitempty"`
		Amount               *big.Int              `json:"amount"`
		Asset                string                `json:"asset"`
		Metadata             map[string]string     `json:"metadata"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := PaymentInitiationIDFromString(aux.ID)
	if err != nil {
		return err
	}

	connectorID, err := ConnectorIDFromString(aux.ConnectorID)
	if err != nil {
		return err
	}

	var sourceAccountID *AccountID
	if aux.SourceAccountID != nil {
		id, err := AccountIDFromString(*aux.SourceAccountID)
		if err != nil {
			return err
		}
		sourceAccountID = &id
	}

	var destinationAccountID *AccountID
	if aux.DestinationAccountID != nil {
		id, err := AccountIDFromString(*aux.DestinationAccountID)
		if err != nil {
			return err
		}
		destinationAccountID = &id
	}

	pi.ID = id
	pi.ConnectorID = connectorID
	pi.Reference = aux.Reference
	pi.CreatedAt = aux.CreatedAt
	pi.ScheduledAt = aux.ScheduledAt
	pi.Description = aux.Description
	pi.Type = aux.Type
	pi.SourceAccountID = sourceAccountID
	pi.DestinationAccountID = destinationAccountID
	pi.Amount = aux.Amount
	pi.Asset = aux.Asset
	pi.Metadata = aux.Metadata

	return nil
}

func FromPaymentInitiationToPSPPaymentInitiation(from *PaymentInitiation, sourceAccount, destinationAccount *PSPAccount) PSPPaymentInitiation {
	return PSPPaymentInitiation{
		Reference:          from.Reference,
		CreatedAt:          from.ScheduledAt, // Scheduled at should be the creation time of the payment on the PSP
		Description:        from.Description,
		SourceAccount:      sourceAccount,
		DestinationAccount: destinationAccount,
		Amount:             from.Amount,
		Asset:              from.Asset,
		Metadata:           from.Metadata,
	}
}

type PaymentInitiationExpanded struct {
	PaymentInitiation PaymentInitiation
	Status            PaymentInitiationAdjustmentStatus
	Error             error
}

func (pi PaymentInitiationExpanded) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                   string                `json:"id"`
		ConnectorID          string                `json:"connectorID"`
		Reference            string                `json:"reference"`
		CreatedAt            time.Time             `json:"createdAt"`
		ScheduledAt          time.Time             `json:"scheduledAt"`
		Description          string                `json:"description"`
		Type                 PaymentInitiationType `json:"type"`
		SourceAccountID      *string               `json:"sourceAccountID,omitempty"`
		DestinationAccountID *string               `json:"destinationAccountID,omitempty"`
		Amount               *big.Int              `json:"amount"`
		Asset                string                `json:"asset"`
		Metadata             map[string]string     `json:"metadata"`
		Status               string                `json:"status"`
		Error                *string               `json:"error,omitempty"`
	}{
		ID:          pi.PaymentInitiation.ID.String(),
		ConnectorID: pi.PaymentInitiation.ConnectorID.String(),
		Reference:   pi.PaymentInitiation.Reference,
		CreatedAt:   pi.PaymentInitiation.CreatedAt,
		ScheduledAt: pi.PaymentInitiation.ScheduledAt,
		Description: pi.PaymentInitiation.Description,
		Type:        pi.PaymentInitiation.Type,
		SourceAccountID: func() *string {
			if pi.PaymentInitiation.SourceAccountID == nil {
				return nil
			}
			return pointer.For(pi.PaymentInitiation.SourceAccountID.String())
		}(),
		DestinationAccountID: func() *string {
			if pi.PaymentInitiation.DestinationAccountID == nil {
				return nil
			}
			return pointer.For(pi.PaymentInitiation.DestinationAccountID.String())
		}(),
		Amount:   pi.PaymentInitiation.Amount,
		Asset:    pi.PaymentInitiation.Asset,
		Metadata: pi.PaymentInitiation.Metadata,
		Status:   pi.Status.String(),
		Error: func() *string {
			if pi.Error == nil {
				return nil
			}
			return pointer.For(pi.Error.Error())
		}(),
	})
}
