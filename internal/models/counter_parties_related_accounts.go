package models

import (
	"encoding/json"
	"time"
)

type CounterPartiesRelatedAccount struct {
	AccountID AccountID `json:"accountID"`
	CreatedAt time.Time `json:"createdAt"`
}

func (b CounterPartiesRelatedAccount) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		AccountID string    `json:"accountID"`
		CreatedAt time.Time `json:"createdAt"`
	}{
		AccountID: b.AccountID.String(),
		CreatedAt: b.CreatedAt,
	})
}

func (b *CounterPartiesRelatedAccount) UnmarshalJSON(data []byte) error {
	var aux struct {
		AccountID string    `json:"accountID"`
		CreatedAt time.Time `json:"createdAt"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	accountID, err := AccountIDFromString(aux.AccountID)
	if err != nil {
		return err
	}

	b.AccountID = accountID
	b.CreatedAt = aux.CreatedAt

	return nil
}
