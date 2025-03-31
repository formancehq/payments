// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"math/big"
	"time"
)

type PaymentAdjustmentRaw struct {
}

type PaymentAdjustment struct {
	Reference string               `json:"reference"`
	CreatedAt time.Time            `json:"createdAt"`
	Status    PaymentStatus        `json:"status"`
	Amount    *big.Int             `json:"amount"`
	Raw       PaymentAdjustmentRaw `json:"raw"`
}

func (p PaymentAdjustment) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(p, "", false)
}

func (p *PaymentAdjustment) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &p, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *PaymentAdjustment) GetReference() string {
	if o == nil {
		return ""
	}
	return o.Reference
}

func (o *PaymentAdjustment) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *PaymentAdjustment) GetStatus() PaymentStatus {
	if o == nil {
		return PaymentStatus("")
	}
	return o.Status
}

func (o *PaymentAdjustment) GetAmount() *big.Int {
	if o == nil {
		return big.NewInt(0)
	}
	return o.Amount
}

func (o *PaymentAdjustment) GetRaw() PaymentAdjustmentRaw {
	if o == nil {
		return PaymentAdjustmentRaw{}
	}
	return o.Raw
}
