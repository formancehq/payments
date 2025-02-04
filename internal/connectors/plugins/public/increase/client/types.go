package client

type Account struct {
	ID        string
	Name      string
	Type      string
	Status    string
	Currency  string
	Balance   int64
	CreatedAt string
}

type Balance struct {
	AccountID     string
	Currency      string
	Amount        int64
	LastUpdatedAt string
}

type ExternalAccount struct {
	ID            string
	Name          string
	Type          string
	Status        string
	Currency      string
	AccountNumber string
	RoutingNumber string
}

type Transaction struct {
	ID          string
	Type        string
	Status      string
	Amount      int64
	Currency    string
	Description string
	CreatedAt   string
}

type TransferRequest struct {
	SourceAccountID      string
	DestinationAccountID string
	Amount               int64
	Currency             string
	Description          string
}

type TransferResponse struct {
	ID          string
	Status      string
	Amount      int64
	Currency    string
	Description string
	CreatedAt   string
}

type PayoutRequest struct {
	AccountID   string
	Amount      int64
	Currency    string
	Description string
}

type PayoutResponse struct {
	ID          string
	Status      string
	Amount      int64
	Currency    string
	Description string
	CreatedAt   string
}
