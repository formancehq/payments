package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
)

type PaymentInitiationReversalAdjustment struct {
	// Unique ID
	ID PaymentInitiationReversalAdjustmentID `json:"id"`

	// Related Payment Initiation Reversal ID
	PaymentInitiationReversalID PaymentInitiationReversalID `json:"paymentInitiationReversalID"`
	// Creation date of the adjustment
	CreatedAt time.Time `json:"createdAt"`
	// Last status of the adjustment
	Status PaymentInitiationReversalAdjustmentStatus `json:"status"`
	// Error description if we had one
	Error error `json:"error"`
	// Additional metadata
	Metadata map[string]string `json:"metadata"`
}

func (piara PaymentInitiationReversalAdjustment) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                          string                                    `json:"id"`
		PaymentInitiationReversalID string                                    `json:"paymentInitiationReversalID"`
		CreatedAt                   time.Time                                 `json:"createdAt"`
		Status                      PaymentInitiationReversalAdjustmentStatus `json:"status"`
		Error                       *string                                   `json:"error,omitempty"`
		Metadata                    map[string]string                         `json:"metadata"`
	}{
		ID:                          piara.ID.String(),
		PaymentInitiationReversalID: piara.PaymentInitiationReversalID.String(),
		CreatedAt:                   piara.CreatedAt,
		Status:                      piara.Status,
		Error: func() *string {
			if piara.Error == nil {
				return nil
			}

			return pointer.For(piara.Error.Error())
		}(),
		Metadata: piara.Metadata,
	})
}

func (piara *PaymentInitiationReversalAdjustment) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                          string                                    `json:"id"`
		PaymentInitiationReversalID string                                    `json:"paymentInitiationReversalID"`
		CreatedAt                   time.Time                                 `json:"createdAt"`
		Status                      PaymentInitiationReversalAdjustmentStatus `json:"status"`
		Error                       *string                                   `json:"error"`
		Metadata                    map[string]string                         `json:"metadata"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := PaymentInitiationReversalAdjustmentIDFromString(aux.ID)
	if err != nil {
		return err
	}

	reversalID, err := PaymentInitiationReversalIDFromString(aux.PaymentInitiationReversalID)
	if err != nil {
		return err
	}

	piara.ID = id
	piara.PaymentInitiationReversalID = reversalID
	piara.CreatedAt = aux.CreatedAt
	piara.Status = aux.Status
	piara.Metadata = aux.Metadata

	if aux.Error != nil {
		piara.Error = errors.New(*aux.Error)
	}

	return nil
}
