// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"math/big"
	"time"
)

type PaymentRaw struct {
}

type Payment struct {
	ID                   string              `json:"id"`
	Reference            string              `json:"reference"`
	SourceAccountID      string              `json:"sourceAccountID"`
	DestinationAccountID string              `json:"destinationAccountID"`
	ConnectorID          string              `json:"connectorID"`
	Provider             *Connector          `json:"provider,omitempty"`
	Type                 PaymentType         `json:"type"`
	Status               PaymentStatus       `json:"status"`
	InitialAmount        *big.Int            `json:"initialAmount"`
	Amount               *big.Int            `json:"amount"`
	Scheme               PaymentScheme       `json:"scheme"`
	Asset                string              `json:"asset"`
	CreatedAt            time.Time           `json:"createdAt"`
	Raw                  *PaymentRaw         `json:"raw"`
	Adjustments          []PaymentAdjustment `json:"adjustments"`
	Metadata             map[string]string   `json:"metadata"`
}

func (p Payment) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(p, "", false)
}

func (p *Payment) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &p, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *Payment) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *Payment) GetReference() string {
	if o == nil {
		return ""
	}
	return o.Reference
}

func (o *Payment) GetSourceAccountID() string {
	if o == nil {
		return ""
	}
	return o.SourceAccountID
}

func (o *Payment) GetDestinationAccountID() string {
	if o == nil {
		return ""
	}
	return o.DestinationAccountID
}

func (o *Payment) GetConnectorID() string {
	if o == nil {
		return ""
	}
	return o.ConnectorID
}

func (o *Payment) GetProvider() *Connector {
	if o == nil {
		return nil
	}
	return o.Provider
}

func (o *Payment) GetType() PaymentType {
	if o == nil {
		return PaymentType("")
	}
	return o.Type
}

func (o *Payment) GetStatus() PaymentStatus {
	if o == nil {
		return PaymentStatus("")
	}
	return o.Status
}

func (o *Payment) GetInitialAmount() *big.Int {
	if o == nil {
		return big.NewInt(0)
	}
	return o.InitialAmount
}

func (o *Payment) GetAmount() *big.Int {
	if o == nil {
		return big.NewInt(0)
	}
	return o.Amount
}

func (o *Payment) GetScheme() PaymentScheme {
	if o == nil {
		return PaymentScheme("")
	}
	return o.Scheme
}

func (o *Payment) GetAsset() string {
	if o == nil {
		return ""
	}
	return o.Asset
}

func (o *Payment) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *Payment) GetRaw() *PaymentRaw {
	if o == nil {
		return nil
	}
	return o.Raw
}

func (o *Payment) GetAdjustments() []PaymentAdjustment {
	if o == nil {
		return []PaymentAdjustment{}
	}
	return o.Adjustments
}

func (o *Payment) GetMetadata() map[string]string {
	if o == nil {
		return nil
	}
	return o.Metadata
}
