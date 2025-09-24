package client

import (
	"encoding/json"
	"time"
)

type PaginationLinks struct {
	Self struct {
		Href string `json:"href"`
	} `json:"self"`
	Next struct {
		Href string `json:"href"`
	} `json:"next"`
	Prev struct {
		Href string `json:"href"`
	} `json:"prev"`
}

type Transaction struct {
	ID         int         `json:"id"`
	AccountID  int         `json:"id_account"`
	Date       time.Time   `json:"date"`
	DateTime   time.Time   `json:"date_time"`
	Value      json.Number `json:"value"`
	Type       string      `json:"type"`
	LastUpdate time.Time   `json:"last_update"`
}

func (t Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID         int         `json:"id"`
		AccountID  int         `json:"id_account"`
		Date       string      `json:"date"`
		DateTime   string      `json:"date_time"`
		Value      json.Number `json:"value"`
		Type       string      `json:"type"`
		LastUpdate string      `json:"last_update"`
	}{
		ID:         t.ID,
		AccountID:  t.AccountID,
		Date:       t.Date.Format(time.DateOnly),
		DateTime:   t.DateTime.Format(time.RFC3339),
		Value:      t.Value,
		Type:       t.Type,
		LastUpdate: t.LastUpdate.Format(time.DateTime),
	})
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	var err error
	type transaction struct {
		ID         int         `json:"id"`
		AccountID  int         `json:"id_account"`
		Date       string      `json:"date"`
		DateTime   string      `json:"date_time"`
		Value      json.Number `json:"value"`
		Type       string      `json:"type"`
		LastUpdate string      `json:"last_update"`
	}

	var tr transaction
	if err := json.Unmarshal(data, &tr); err != nil {
		return err
	}

	t.ID = tr.ID
	t.AccountID = tr.AccountID

	if tr.Date != "" {
		t.Date, err = time.Parse(time.DateOnly, tr.Date)
		if err != nil {
			return err
		}
	}

	if tr.DateTime != "" {
		t.DateTime, err = time.Parse(time.RFC3339, tr.DateTime)
		if err != nil {
			return err
		}
	}
	t.Value = tr.Value
	t.Type = tr.Type

	if tr.LastUpdate != "" {
		t.LastUpdate, err = time.Parse(time.DateTime, tr.LastUpdate)
		if err != nil {
			return err
		}
	}

	return nil
}
