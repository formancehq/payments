package client

import "time"

type Account struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OpeningDate time.Time `json:"opening_date"`
	Currency    string    `json:"currency"`
}
