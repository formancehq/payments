package models

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/google/uuid"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/utils/assets"
)

// Internal struct used by the plugins
type PSPPayment struct {
	// Original PSP payment/transaction reference.
	// In case of refunds, dispute etc... this reference should be the original
	// payment reference. In case it's the first payment, ths reference should
	// be empty
	ParentReference string

	// PSP payment/transaction reference. Should be unique.
	Reference string

	// Payment Creation date.
	CreatedAt time.Time

	// Type of payment: payin, payout, transfer etc...
	Type PaymentType

	// Payment amount.
	Amount *big.Int

	// Currency. Should be in minor currencies unit.
	// For example: USD/2
	Asset string

	// Payment scheme if existing: visa, mastercard etc...
	Scheme PaymentScheme

	// Payment status: pending, failed, succeeded etc...
	Status PaymentStatus

	// Optional, can be filled for payouts and transfers for example.
	SourceAccountReference *string
	// Optional, can be filled for payins and transfers for example.
	DestinationAccountReference *string

	// Optional, can be filled if the payment is related to an open banking connector
	PsuID *uuid.UUID
	// Optional, can be filled if the payment is related to an open banking connector
	OpenBankingConnectionID *string

	// Additional metadata
	Metadata map[string]string

	// PSP response in raw
	Raw json.RawMessage
}

type PSPPaymentsToDelete struct {
	Reference string
}

type PSPPaymentsToCancel struct {
	Reference string
}

func (p *PSPPayment) Validate() error {
	if p.Reference == "" {
		return errorsutils.NewWrappedError(errors.New("missing payment reference"), ErrValidation)
	}

	if p.CreatedAt.IsZero() {
		return errorsutils.NewWrappedError(errors.New("missing payment createdAt"), ErrValidation)
	}

	if p.Type == PAYMENT_TYPE_UNKNOWN {
		return errorsutils.NewWrappedError(errors.New("missing payment type"), ErrValidation)
	}

	if p.Amount == nil {
		return errorsutils.NewWrappedError(errors.New("missing payment amount"), ErrValidation)
	}

	if !assets.IsValid(p.Asset) {
		return errorsutils.NewWrappedError(errors.New("invalid payment asset"), ErrValidation)
	}

	if p.Raw == nil {
		return errorsutils.NewWrappedError(errors.New("missing payment raw"), ErrValidation)
	}

	return nil
}

func (p *PSPPayment) HasParent() bool {
	return p.ParentReference != ""
}

type Payment struct {
	// Unique Payment ID generated from payments information
	ID PaymentID `json:"id"`
	// Related Connector ID
	ConnectorID ConnectorID `json:"connectorID"`

	// PSP payment/transaction reference. Should be unique.
	Reference string `json:"reference"`

	// Payment Creation date.
	CreatedAt time.Time `json:"createdAt"`

	// Type of payment: payin, payout, transfer etc...
	Type PaymentType `json:"type"`

	// Payment Initial amount
	InitialAmount *big.Int `json:"initialAmount"`
	// Payment amount.
	Amount *big.Int `json:"amount"`

	// Currency. Should be in minor currencies unit.
	// For example: USD/2
	Asset string `json:"asset"`

	// Payment scheme if existing: visa, mastercard etc...
	Scheme PaymentScheme `json:"scheme"`

	// Payment status: pending, failed, succeeded etc...
	Status PaymentStatus `json:"status"`

	// Optional, can be filled for payouts and transfers for example.
	SourceAccountID *AccountID `json:"sourceAccountID"`
	// Optional, can be filled for payins and transfers for example.
	DestinationAccountID *AccountID `json:"destinationAccountID"`
	// Optional, can be filled if the payment is related to an open banking connector
	PsuID *uuid.UUID `json:"psuID"`
	// Optional, can be filled if the payment is related to an open banking connector
	OpenBankingConnectionID *string `json:"openBankingConnectionID"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`

	// Related adjustment
	Adjustments []PaymentAdjustment `json:"adjustments"`
}

func (p Payment) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                      string              `json:"id"`
		ConnectorID             string              `json:"connectorID"`
		Provider                string              `json:"provider"`
		Reference               string              `json:"reference"`
		CreatedAt               time.Time           `json:"createdAt"`
		Type                    PaymentType         `json:"type"`
		InitialAmount           *big.Int            `json:"initialAmount"`
		Amount                  *big.Int            `json:"amount"`
		Asset                   string              `json:"asset"`
		Scheme                  PaymentScheme       `json:"scheme"`
		Status                  PaymentStatus       `json:"status"`
		SourceAccountID         *string             `json:"sourceAccountID"`
		DestinationAccountID    *string             `json:"destinationAccountID"`
		PsuID                   *string             `json:"psuID,omitempty"`
		OpenBankingConnectionID *string             `json:"openBankingConnectionID,omitempty"`
		Metadata                map[string]string   `json:"metadata"`
		Adjustments             []PaymentAdjustment `json:"adjustments"`
	}{
		ID:            p.ID.String(),
		ConnectorID:   p.ConnectorID.String(),
		Provider:      ToV3Provider(p.ConnectorID.Provider),
		Reference:     p.Reference,
		CreatedAt:     p.CreatedAt,
		Type:          p.Type,
		InitialAmount: p.InitialAmount,
		Amount:        p.Amount,
		Asset:         p.Asset,
		Scheme:        p.Scheme,
		Status:        p.Status,
		SourceAccountID: func() *string {
			if p.SourceAccountID == nil {
				return nil
			}
			return pointer.For(p.SourceAccountID.String())
		}(),
		DestinationAccountID: func() *string {
			if p.DestinationAccountID == nil {
				return nil
			}
			return pointer.For(p.DestinationAccountID.String())
		}(),
		PsuID: func() *string {
			if p.PsuID == nil {
				return nil
			}
			return pointer.For(p.PsuID.String())
		}(),
		OpenBankingConnectionID: p.OpenBankingConnectionID,
		Metadata:                p.Metadata,
		Adjustments:             p.Adjustments,
	})
}

func (c *Payment) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                      string              `json:"id"`
		ConnectorID             string              `json:"connectorID"`
		Provider                string              `json:"provider"`
		Reference               string              `json:"reference"`
		CreatedAt               time.Time           `json:"createdAt"`
		Type                    PaymentType         `json:"type"`
		InitialAmount           *big.Int            `json:"initialAmount"`
		Amount                  *big.Int            `json:"amount"`
		Asset                   string              `json:"asset"`
		Scheme                  PaymentScheme       `json:"scheme"`
		Status                  PaymentStatus       `json:"status"`
		SourceAccountID         *string             `json:"sourceAccountID"`
		DestinationAccountID    *string             `json:"destinationAccountID"`
		PsuID                   *string             `json:"psuID,omitempty"`
		OpenBankingConnectionID *string             `json:"openBankingConnectionID,omitempty"`
		Metadata                map[string]string   `json:"metadata"`
		Adjustments             []PaymentAdjustment `json:"adjustments"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := PaymentIDFromString(aux.ID)
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

	if aux.PsuID != nil {
		psuID, err := uuid.Parse(*aux.PsuID)
		if err != nil {
			return err
		}
		c.PsuID = &psuID
	}

	if aux.OpenBankingConnectionID != nil {
		c.OpenBankingConnectionID = aux.OpenBankingConnectionID
	}

	c.ID = id
	c.ConnectorID = connectorID
	c.Reference = aux.Reference
	c.CreatedAt = aux.CreatedAt
	c.Type = aux.Type
	c.InitialAmount = aux.InitialAmount
	c.Amount = aux.Amount
	c.Asset = aux.Asset
	c.Scheme = aux.Scheme
	c.Status = aux.Status
	c.SourceAccountID = sourceAccountID
	c.DestinationAccountID = destinationAccountID
	c.Metadata = aux.Metadata
	c.Adjustments = aux.Adjustments

	return nil
}

func FromPSPPaymentToPayment(from PSPPayment, connectorID ConnectorID) (Payment, error) {
	if err := from.Validate(); err != nil {
		return Payment{}, err
	}

	paymentReference := from.Reference
	if from.HasParent() {
		paymentReference = from.ParentReference
	}

	p := Payment{
		ID: PaymentID{
			PaymentReference: PaymentReference{
				Reference: paymentReference,
				Type:      from.Type,
			},
			ConnectorID: connectorID,
		},
		ConnectorID:   connectorID,
		Reference:     paymentReference,
		CreatedAt:     from.CreatedAt,
		Type:          from.Type,
		InitialAmount: from.Amount,
		Amount:        from.Amount,
		Asset:         from.Asset,
		Scheme:        from.Scheme,
		Status:        from.Status,
		SourceAccountID: func() *AccountID {
			if from.SourceAccountReference == nil {
				return nil
			}
			return &AccountID{
				Reference:   *from.SourceAccountReference,
				ConnectorID: connectorID,
			}
		}(),
		DestinationAccountID: func() *AccountID {
			if from.DestinationAccountReference == nil {
				return nil
			}
			return &AccountID{
				Reference:   *from.DestinationAccountReference,
				ConnectorID: connectorID,
			}
		}(),
		PsuID: func() *uuid.UUID {
			if from.PsuID == nil {
				return nil
			}
			return from.PsuID
		}(),
		OpenBankingConnectionID: from.OpenBankingConnectionID,
		Metadata:                from.Metadata,
	}

	if p.Status == PAYMENT_STATUS_AUTHORISATION {
		// Will be capture later
		p.Amount = big.NewInt(0)
	}

	p.Adjustments = append(p.Adjustments, FromPSPPaymentToPaymentAdjustment(from, connectorID))

	return p, nil
}

func FromPSPPayments(from []PSPPayment, connectorID ConnectorID) ([]Payment, error) {
	payments := make([]Payment, 0, len(from))
	for _, p := range from {
		payment, err := FromPSPPaymentToPayment(p, connectorID)
		if err != nil {
			return nil, err
		}

		payments = append(payments, payment)
	}
	return payments, nil
}

func FromPSPPaymentToPaymentAdjustment(from PSPPayment, connectorID ConnectorID) PaymentAdjustment {
	parentReference := from.Reference
	if from.HasParent() {
		parentReference = from.ParentReference
	}

	paymentID := PaymentID{
		PaymentReference: PaymentReference{
			Reference: parentReference,
			Type:      from.Type,
		},
		ConnectorID: connectorID,
	}

	return PaymentAdjustment{
		ID: PaymentAdjustmentID{
			PaymentID: paymentID,
			Reference: from.Reference,
			CreatedAt: from.CreatedAt,
			Status:    from.Status,
		},
		Reference: from.Reference,
		CreatedAt: from.CreatedAt,
		Status:    from.Status,
		Amount:    from.Amount,
		Asset:     &from.Asset,
		Metadata:  from.Metadata,
		Raw:       from.Raw,
	}
}
