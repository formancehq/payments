package models

import (
	"time"

	"github.com/google/uuid"
)

type Pool struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`

	PoolAccounts []AccountID `json:"poolAccounts"`
}

func (p *Pool) IdempotencyKey() string {
	relatedAccounts := make([]string, len(p.PoolAccounts))
	for i := range p.PoolAccounts {
		relatedAccounts[i] = p.PoolAccounts[i].String()
	}
	var ik = struct {
		ID              string
		RelatedAccounts []string
	}{
		ID:              p.ID.String(),
		RelatedAccounts: relatedAccounts,
	}
	return IdempotencyKey(ik)
}
