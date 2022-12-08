package models

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Adjustment struct {
	bun.BaseModel `bun:"payments.adjustment"`

	ID        uuid.UUID
	PaymentID uuid.UUID
	CreatedAt time.Time
	Amount    int64
	Status    PaymentStatus
	Absolute  bool

	RawData any

	Payment *Payment `bun:"rel:has-one,join:payment_id=id"`
}
