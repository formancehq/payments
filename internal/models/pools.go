package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PoolType string

const (
	// Pools created with a list of accounts. No dynamic changes, the user can
	// add/delete accounts from the pool via specific endpoints.
	POOL_TYPE_STATIC PoolType = "STATIC"
	// Pools created with an account list query. The user cannot add/delete
	// accounts from the pool directly from endpoints, but can change the query
	// to match the right accounts.
	POOL_TYPE_DYNAMIC PoolType = "DYNAMIC"
)

type Pool struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Type      PoolType  `json:"type"`

	Query map[string]any `json:"query"`

	PoolAccounts []AccountID `json:"poolAccounts"`
}

func (p Pool) MarshalJSON() ([]byte, error) {
	var aux struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		CreatedAt    time.Time `json:"createdAt"`
		Type         PoolType  `json:"type"`
		PoolAccounts []string  `json:"poolAccounts"`
	}

	aux.ID = p.ID.String()
	aux.Name = p.Name
	aux.CreatedAt = p.CreatedAt
	aux.Type = p.Type

	aux.PoolAccounts = make([]string, len(p.PoolAccounts))
	for i := range p.PoolAccounts {
		aux.PoolAccounts[i] = p.PoolAccounts[i].String()
	}

	return json.Marshal(aux)
}

func (p *Pool) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		CreatedAt    time.Time `json:"createdAt"`
		Type         PoolType  `json:"type"`
		PoolAccounts []string  `json:"poolAccounts"`
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
	p.Type = aux.Type
	p.PoolAccounts = poolAccounts

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
	}{
		ID:              p.ID.String(),
		RelatedAccounts: relatedAccounts,
	}
	return IdempotencyKey(ik)
}
