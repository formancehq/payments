package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID        uuid.UUID `json:"id,omitempty"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (t Token) MarshalJSON() ([]byte, error) {
	type res struct {
		ID        string    `json:"id,omitempty"`
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expiresAt"`
	}

	r := res{
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
	}

	if t.ID != uuid.Nil {
		r.ID = t.ID.String()
	}

	return json.Marshal(r)
}

func (t *Token) UnmarshalJSON(data []byte) error {
	type res struct {
		ID        string    `json:"id,omitempty"`
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expiresAt"`
	}

	var r res
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	t.Token = r.Token
	t.ExpiresAt = r.ExpiresAt

	if r.ID != "" {
		var err error
		t.ID, err = uuid.Parse(r.ID)
		if err != nil {
			return err
		}
	}

	return nil
}
