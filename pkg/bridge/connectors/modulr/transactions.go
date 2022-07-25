package modulr

import (
	"encoding/json"
	"fmt"
)

type Transaction struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Amount          float64 `json:"amount"`
	Credit          bool    `json:"credit"`
	SourceID        string  `json:"sourceId"`
	Description     string  `json:"description"`
	PostedDate      string  `json:"postedDate"`
	TransactionDate string  `json:"transactionDate"`
	Account         Account `json:"account"`
	AdditionalInfo  struct {
		Payer struct {
			Name       string `json:"name"`
			Identifier struct {
				Type          string `json:"type"`
				AccountNumber string `json:"accountNumber"`
				SortCode      string `json:"sortCode"`
			}
		} `json:"payer"`
	} `json:"additionalInfo"`
}

func (m *ModulrClient) GetTransactions(accountId string) ([]Transaction, error) {
	resp, err := m.httpClient.Get(m.Endpoint("accounts/%s/transactions", accountId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var res ResponseWrapper[[]Transaction]
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res.Content, nil

}
