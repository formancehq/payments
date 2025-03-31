// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"time"
)

type TransferInitiationPayments struct {
	PaymentID string        `json:"paymentID"`
	CreatedAt time.Time     `json:"createdAt"`
	Status    PaymentStatus `json:"status"`
	Error     *string       `json:"error,omitempty"`
}

func (t TransferInitiationPayments) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(t, "", false)
}

func (t *TransferInitiationPayments) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &t, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *TransferInitiationPayments) GetPaymentID() string {
	if o == nil {
		return ""
	}
	return o.PaymentID
}

func (o *TransferInitiationPayments) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *TransferInitiationPayments) GetStatus() PaymentStatus {
	if o == nil {
		return PaymentStatus("")
	}
	return o.Status
}

func (o *TransferInitiationPayments) GetError() *string {
	if o == nil {
		return nil
	}
	return o.Error
}
