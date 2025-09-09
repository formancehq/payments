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

	Transactions []Transaction `json:"transactions"`
}

func (b BankAccount) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           int      `json:"id"`
		UserID       int      `json:"id_user"`
		ConnectionID int      `json:"id_connection"`
		Currency     Currency `json:"currency"`
		OriginalName string   `json:"original_name"`
		Error        string   `json:"error"`
		LastUpdate   string   `json:"last_update"`

		Transactions []Transaction `json:"transactions"`
	}{
		ID:           b.ID,
		UserID:       b.UserID,
		ConnectionID: b.ConnectionID,
		Currency:     b.Currency,
		OriginalName: b.OriginalName,
		Error:        b.Error,
		LastUpdate:   b.LastUpdate.Format(time.DateTime),

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
		LastUpdate   string   `json:"last_update"`

		Transactions []Transaction `json:"transactions"`
	}

	var ba bankAccount
	if err := json.Unmarshal(data, &ba); err != nil {
		return err
	}

	var lastUpdate time.Time
	if ba.LastUpdate != "" {
		var err error
		lastUpdate, err = time.Parse(time.DateTime, ba.LastUpdate)
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
		Transactions: ba.Transactions,
	}

	return nil
}
