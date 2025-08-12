package client

type Currency struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Precision int    `json:"precision"`
}

type BankAccount struct {
	ID           int      `json:"id"`
	ConnectionID int      `json:"id_connection"`
	Currency     Currency `json:"currency"`
	OriginalName string   `json:"original_name"`
	Error        string   `json:"error"`
}
