package client

type Balance struct {
	AccountID      string `json:"account_id"`
	AmountInMinors int64  `json:"amount_in_minors"`
	Currency       string `json:"currency"`
}
