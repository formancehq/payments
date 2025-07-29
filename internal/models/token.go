package models

import (
	"encoding/json"
	"time"
)

type Token struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (t Token) MarshalJSON() ([]byte, error) {
	type res struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expiresAt"`
	}

	r := res{
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
	}

	return json.Marshal(r)
}

func (t *Token) UnmarshalJSON(data []byte) error {
	type res struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expiresAt"`
	}

	var r res
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	t.Token = r.Token
	t.ExpiresAt = r.ExpiresAt

	return nil
}
