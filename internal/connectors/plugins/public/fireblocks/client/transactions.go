package client

import (
	"context"
	"fmt"
	"net/http"
)

type Transaction struct {
	ID          string       `json:"id"`
	AssetID     string       `json:"assetId"`
	Source      TransferPeer `json:"source"`
	Destination TransferPeer `json:"destination"`
	AmountInfo  AmountInfo   `json:"amountInfo"`
	FeeInfo     FeeInfo      `json:"feeInfo"`
	Operation   string       `json:"operation"`
	Status      string       `json:"status"`
	SubStatus   string       `json:"subStatus"`
	TxHash      string       `json:"txHash"`
	CreatedAt   int64        `json:"createdAt"`
	LastUpdated int64        `json:"lastUpdated"`
}

type TransferPeer struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	SubType string `json:"subType"`
}

type AmountInfo struct {
	Amount          string `json:"amount"`
	RequestedAmount string `json:"requestedAmount"`
	NetAmount       string `json:"netAmount"`
	AmountUSD       string `json:"amountUSD"`
}

type FeeInfo struct {
	NetworkFee string `json:"networkFee"`
	ServiceFee string `json:"serviceFee"`
	GasPrice   string `json:"gasPrice"`
}

func (c *client) ListTransactions(ctx context.Context, createdAfter int64, limit int) ([]Transaction, error) {
	endpoint := fmt.Sprintf("%s/v1/transactions?limit=%d&orderBy=createdAt&sort=ASC", c.baseURL, limit)
	if createdAfter > 0 {
		endpoint = fmt.Sprintf("%s&after=%d", endpoint, createdAfter)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var response []Transaction
	var errResponse fireblocksError
	_, err = c.httpClient.Do(ctx, req, &response, &errResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return response, nil
}
