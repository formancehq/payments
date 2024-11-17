package models

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
)

type PSPPaymentInitiationReversal struct {
	// Reference of the unique payment initiation reversal
	Reference string

	// Payment Initiation Reversal creation date
	CreatedAt time.Time

	// Description of the payment initiation reversal
	Description string

	// Related Payment Initiation object
	RelatedPaymentInitiation PSPPaymentInitiation

	// Amount that we want to reverse
	Amount *big.Int
	// Asset of the reversal
	Asset string

	// Additional metadata
	Metadata map[string]string
}

type PaymentInitiationReversal struct {
	// Unique Payment initiation reversal ID generated from reversal information
	ID PaymentInitiationReversalID `json:"id"`
	// Related Connector ID
	ConnectorID ConnectorID `json:"connectorID"`
	// Related Payment Initiation ID that is being reversed
	PaymentInitiationID PaymentInitiationID `json:"paymentInitiationID"`
	// Unique reference of the reversal
	Reference string `json:"reference"`

	// Payment Initiation Reversal creation date
	CreatedAt time.Time `json:"createdAt"`
	// Description of the reversal
	Description string `json:"description"`

	// Amount reversed
	Amount *big.Int `json:"amount"`
	// Asset of the reversal
	Asset string `json:"asset"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`
}

func (pir PaymentInitiationReversal) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                  string            `json:"id"`
		ConnectorID         string            `json:"connectorID"`
		PaymentInitiationID string            `json:"paymentInitiationID"`
		Reference           string            `json:"reference"`
		CreatedAt           time.Time         `json:"createdAt"`
		Description         string            `json:"description"`
		Amount              *big.Int          `json:"amount"`
		Asset               string            `json:"asset"`
		Metadata            map[string]string `json:"metadata"`
	}{
		ID:                  pir.ID.String(),
		ConnectorID:         pir.ConnectorID.String(),
		PaymentInitiationID: pir.PaymentInitiationID.String(),
		Reference:           pir.Reference,
		CreatedAt:           pir.CreatedAt,
		Description:         pir.Description,
		Amount:              pir.Amount,
		Asset:               pir.Asset,
		Metadata:            pir.Metadata,
	})
}

func (pir *PaymentInitiationReversal) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                  string            `json:"id"`
		ConnectorID         string            `json:"connectorID"`
		PaymentInitiationID string            `json:"paymentInitiationID"`
		Reference           string            `json:"reference"`
		CreatedAt           time.Time         `json:"createdAt"`
		Description         string            `json:"description"`
		Amount              *big.Int          `json:"amount"`
		Asset               string            `json:"asset"`
		Metadata            map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := PaymentInitiationReversalIDFromString(aux.ID)
	if err != nil {
		return err
	}
	connectorID, err := ConnectorIDFromString(aux.ConnectorID)
	if err != nil {
		return err
	}
	paymentInitiationID, err := PaymentInitiationIDFromString(aux.PaymentInitiationID)
	if err != nil {
		return err
	}

	pir.ID = id
	pir.ConnectorID = connectorID
	pir.PaymentInitiationID = paymentInitiationID
	pir.Reference = aux.Reference
	pir.CreatedAt = aux.CreatedAt
	pir.Description = aux.Description
	pir.Amount = aux.Amount
	pir.Asset = aux.Asset
	pir.Metadata = aux.Metadata

	return nil
}

func FromPaymentInitiationReversalToPSPPaymentInitiationReversal(from *PaymentInitiationReversal, relatedPI PSPPaymentInitiation) PSPPaymentInitiationReversal {
	return PSPPaymentInitiationReversal{
		Reference:                from.Reference,
		CreatedAt:                from.CreatedAt,
		Description:              from.Description,
		RelatedPaymentInitiation: relatedPI,
		Amount:                   from.Amount,
		Asset:                    from.Asset,
		Metadata:                 from.Metadata,
	}
}

type PaymentInitiationReversalExpanded struct {
	PaymentInitiationReversal PaymentInitiationReversal
	Status                    PaymentInitiationReversalAdjustmentStatus
	Error                     error
}

func (pi PaymentInitiationReversalExpanded) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                  string            `json:"id"`
		ConnectorID         string            `json:"connectorID"`
		PaymentInitiationID string            `json:"paymentInitiationID"`
		Reference           string            `json:"reference"`
		CreatedAt           time.Time         `json:"createdAt"`
		Description         string            `json:"description"`
		Amount              *big.Int          `json:"amount"`
		Asset               string            `json:"asset"`
		Metadata            map[string]string `json:"metadata"`
		Status              string            `json:"status"`
		Error               *string           `json:"error,omitempty"`
	}{
		ID:                  pi.PaymentInitiationReversal.ID.String(),
		ConnectorID:         pi.PaymentInitiationReversal.ConnectorID.String(),
		PaymentInitiationID: pi.PaymentInitiationReversal.PaymentInitiationID.String(),
		Reference:           pi.PaymentInitiationReversal.Reference,
		CreatedAt:           pi.PaymentInitiationReversal.CreatedAt,
		Description:         pi.PaymentInitiationReversal.Description,
		Amount:              pi.PaymentInitiationReversal.Amount,
		Asset:               pi.PaymentInitiationReversal.Asset,
		Metadata:            pi.PaymentInitiationReversal.Metadata,
		Status:              pi.Status.String(),
		Error: func() *string {
			if pi.Error == nil {
				return nil
			}
			return pointer.For(pi.Error.Error())
		}(),
	})
}
