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
	Date       time.Time   `json:"date,omitempty" `
	DateTime   time.Time   `json:"date_time,omitempty"`
	Value      json.Number `json:"value"`
	Type       string      `json:"type"`
	LastUpdate time.Time   `json:"last_update,omitempty"`
}

func (t Transaction) MarshalJSON() ([]byte, error) {
	var (
		trDate       string
		trDateTime   string
		trLastUpdate string
	)

	// Only format and include fields if the underlying time is non-zero, so that
	// omitempty actually omits absent optional fields.
	if !t.Date.IsZero() {
		// Do not apply timezone conversion for Date; format as date-only string from the given time value.
		// Note that this might be incorrect; I assume the timezone is Europe/Paris, but if we try to change it
		// to UTC, we end up with the previous day all the time which is probably worst than setting the timezone properly.
		trDate = t.Date.Format(time.DateOnly)
	}
	if !t.DateTime.IsZero() {
		// This is documented as being in UTC, maybe it's true -- the examples are all null though.
		trDateTime = t.DateTime.Format(time.DateTime)
	}
	if !t.LastUpdate.IsZero() {
		var err error
		trLastUpdate, err = ConvertUTCToPowensTime(t.LastUpdate, time.DateTime)
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(struct {
		ID         int         `json:"id"`
		AccountID  int         `json:"id_account"`
		Date       string      `json:"date,omitempty"`
		DateTime   string      `json:"date_time,omitempty"`
		Value      json.Number `json:"value"`
		Type       string      `json:"type"`
		LastUpdate string      `json:"last_update,omitempty"`
	}{
		ID:         t.ID,
		AccountID:  t.AccountID,
		Date:       trDate,
		DateTime:   trDateTime,
		Value:      t.Value,
		Type:       t.Type,
		LastUpdate: trLastUpdate,
	})
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	type transaction struct {
		ID         int         `json:"id"`
		AccountID  int         `json:"id_account"`
		Date       string      `json:"date,omitempty"`
		DateTime   string      `json:"date_time,omitempty"`
		Value      json.Number `json:"value"`
		Type       string      `json:"type"`
		LastUpdate string      `json:"last_update,omitempty"`
	}

	var tr transaction
	if err := json.Unmarshal(data, &tr); err != nil {
		return err
	}

	t.ID = tr.ID
	t.AccountID = tr.AccountID

	if tr.Date != "" {
		// Do not apply timezone conversion for Date; parse the date-only string directly without shifting timezones.
		date, err := time.Parse(time.DateOnly, tr.Date)
		if err != nil {
			return err
		}
		t.Date = date.UTC()
	}

	if tr.DateTime != "" {
		// This is documented as being in UTC, maybe it's true -- the examples are all null though.
		date, err := time.Parse(time.DateTime, tr.DateTime)
		if err != nil {
			return err
		}
		t.DateTime = date.UTC()
	}
	t.Value = tr.Value
	t.Type = tr.Type

	if tr.LastUpdate != "" {
		date, err := ConvertPowensTimeToUTC(tr.LastUpdate, time.DateTime)
		if err != nil {
			return err
		}
		t.LastUpdate = date
	}

	return nil
}
