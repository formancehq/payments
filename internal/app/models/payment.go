package models

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Payment struct {
	bun.BaseModel `bun:"payments.payment"`

	ID          uuid.UUID
	ConnectorID uuid.UUID
	CreatedAt   time.Time
	Reference   string
	Amount      int64
	Type        PaymentType
	Status      PaymentStatus
	Scheme      PaymentScheme
	Asset       PaymentAsset

	RawData any

	AccountID uuid.UUID

	Account     *Account      `bun:"rel:has-one,join:account_id=id"`
	Adjustments []*Adjustment `bun:"rel:has-many,join:id=payment_id"`
	Metadata    []*Metadata   `bun:"rel:has-many,join:id=payment_id"`
	Connector   *Connector    `bun:"rel:has-one,join:connector_id=id"`
}

type (
	PaymentType   string
	PaymentStatus string
	PaymentScheme string
	PaymentAsset  string
)

const (
	PaymentTypePayIn    PaymentType = "PAY-IN"
	PaymentTypePayOut   PaymentType = "PAYOUT"
	PaymentTypeTransfer PaymentType = "TRANSFER"
	PaymentTypeOther    PaymentType = "OTHER"
)

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusSucceeded PaymentStatus = "SUCCEEDED"
	PaymentStatusCancelled PaymentStatus = "CANCELLED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusOther     PaymentStatus = "OTHER"
)

func (t PaymentType) String() string {
	return string(t)
}

func (t PaymentStatus) String() string {
	return string(t)
}

func (t PaymentScheme) String() string {
	return string(t)
}

func (t PaymentAsset) String() string {
	return string(t)
}
