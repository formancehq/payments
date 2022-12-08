package models

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Metadata struct {
	bun.BaseModel `bun:"payments.metadata"`

	PaymentID uuid.UUID
	CreatedAt time.Time
	Key       string
	Value     string

	Changelog []*MetadataChangelog `bun:"rel:has-many,join:payment_id=payment_id,join:key=key"`
	Payment   *Payment             `bun:"rel:has-one,join:payment_id=id"`
}

type MetadataChangelog struct {
	bun.BaseModel `bun:"payments.metadata_changelog"`

	PaymentID   uuid.UUID
	CreatedAt   time.Time
	Key         string
	ValueBefore string
	ValueAfter  string

	Metadata *Metadata `bun:"rel:has-one,join:payment_id=payment_id,join:key=key"`
	Payment  *Payment  `bun:"rel:has-one,join:payment_id=id"`
}
