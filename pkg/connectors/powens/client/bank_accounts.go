package client

import (
	"encoding/json"
	"time"
)

type Currency struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Precision int    `json:"precision"`
}

type BankAccount struct {
	ID           int       `json:"id"`
	UserID       int       `json:"id_user"`
	ConnectionID int       `json:"id_connection"`
	Currency     Currency  `json:"currency"`
	OriginalName string    `json:"original_name"`
	Error        string    `json:"error"`
	LastUpdate   time.Time `json:"last_update"`

	Balance      json.Number   `json:"balance"`
	Transactions []Transaction `json:"transactions"`
}

func (b BankAccount) MarshalJSON() ([]byte, error) {
	var lastUpdate string
	if !b.LastUpdate.IsZero() {
		var err error
		lastUpdate, err = ConvertUTCToPowensTime(b.LastUpdate, time.DateTime)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(struct {
		ID           int      `json:"id"`
		UserID       int      `json:"id_user"`
		ConnectionID int      `json:"id_connection"`
		Currency     Currency `json:"currency"`
		OriginalName string   `json:"original_name"`
		Error        string   `json:"error"`
		LastUpdate   string   `json:"last_update,omitempty"`

		Balance      json.Number   `json:"balance"`
		Transactions []Transaction `json:"transactions"`
	}{
		ID:           b.ID,
		UserID:       b.UserID,
		ConnectionID: b.ConnectionID,
		Currency:     b.Currency,
		OriginalName: b.OriginalName,
		Error:        b.Error,
		LastUpdate:   lastUpdate,

		Balance:      b.Balance,
		Transactions: b.Transactions,
	})
}

func (b *BankAccount) UnmarshalJSON(data []byte) error {
	type bankAccount struct {
		ID           int      `json:"id"`
		UserID       int      `json:"id_user"`
		ConnectionID int      `json:"id_connection"`
		Currency     Currency `json:"currency"`
		OriginalName string   `json:"original_name"`
		Error        string   `json:"error"`
		LastUpdate   string   `json:"last_update,omitempty"`

		Balance      json.Number   `json:"balance"`
		Transactions []Transaction `json:"transactions"`
	}

	var ba bankAccount
	if err := json.Unmarshal(data, &ba); err != nil {
		return err
	}

	var lastUpdate time.Time
	if ba.LastUpdate != "" {
		var err error
		lastUpdate, err = ConvertPowensTimeToUTC(ba.LastUpdate, time.DateTime)
		if err != nil {
			return err
		}
	}

	*b = BankAccount{
		ID:           ba.ID,
		UserID:       ba.UserID,
		ConnectionID: ba.ConnectionID,
		Currency:     ba.Currency,
		OriginalName: ba.OriginalName,
		Error:        ba.Error,
		LastUpdate:   lastUpdate,
		Balance:      ba.Balance,
		Transactions: ba.Transactions,
	}

	return nil
}
