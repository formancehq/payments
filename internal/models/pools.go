package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Pool struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`

	PoolAccounts []AccountID `json:"poolAccounts"`
	Query        *string    `json:"query,omitempty"`
}

func (p Pool) MarshalJSON() ([]byte, error) {
	var aux struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		CreatedAt    time.Time `json:"createdAt"`
		PoolAccounts []string  `json:"poolAccounts"`
		Query        *string   `json:"query,omitempty"`
	}

	aux.ID = p.ID.String()
	aux.Name = p.Name
	aux.CreatedAt = p.CreatedAt

	aux.PoolAccounts = make([]string, len(p.PoolAccounts))
	for i := range p.PoolAccounts {
		aux.PoolAccounts[i] = p.PoolAccounts[i].String()
	}
	aux.Query = p.Query

	return json.Marshal(aux)
}

func (p *Pool) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		CreatedAt    time.Time `json:"createdAt"`
		PoolAccounts []string  `json:"poolAccounts"`
		Query        *string   `json:"query"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := uuid.Parse(aux.ID)
	if err != nil {
		return err
	}

	poolAccounts := make([]AccountID, len(aux.PoolAccounts))
	for i := range aux.PoolAccounts {
		accID, err := AccountIDFromString(aux.PoolAccounts[i])
		if err != nil {
			return err
		}
		poolAccounts[i] = accID
	}

	p.ID = id
	p.Name = aux.Name
	p.CreatedAt = aux.CreatedAt
	p.PoolAccounts = poolAccounts
	p.Query = aux.Query

	return nil
}

func (p *Pool) IdempotencyKey() string {
	relatedAccounts := make([]string, len(p.PoolAccounts))
	for i := range p.PoolAccounts {
		relatedAccounts[i] = p.PoolAccounts[i].String()
	}
	var ik = struct {
		ID              string
		RelatedAccounts []string
		Query           *string
	}{
		ID:              p.ID.String(),
		RelatedAccounts: relatedAccounts,
		Query:           p.Query,
	}
	return IdempotencyKey(ik)
}
