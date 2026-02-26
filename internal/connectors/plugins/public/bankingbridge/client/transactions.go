package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Transaction struct {
	ID                   string `json:"id"`
	AccountReference     string `json:"accountReference"`
	BookedAt             string `json:"bookedAt"`
	SettledAt            string `json:"settledAt"`
	Status               string `json:"status"`
	Asset                string `json:"asset"`
	AmountInMinors       int64  `json:"amountInMinors"`
	Scheme               string `json:"scheme"`
	IsReversal           bool   `json:"isReversal"`
	IsBatch              bool   `json:"isBatch"`
	NumberOfTransactions int    `json:"numberOfTransactions"`
	EntryReference       string `json:"entryReference"`
	ServicerReference    string `json:"servicerReference"`
	BatchMessageId       string `json:"batchMessageId"`
	BatchPaymentInfoId   string `json:"batchPaymentInfoId"`
	ImportedAt           string `json:"importedAt"`
	UpdatedAt            string `json:"updatedAt"`
}

func (c *client) GetTransactions(ctx context.Context, cursor string, pageSize int) ([]Transaction, bool, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	// TODO: call PSP to get transactions
	return nil, false, "", nil
}
