package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/utils/errors"
)

type Transaction struct {
	ID                   string    `json:"id"`
	AccountReference     string    `json:"accountReference"`
	BookedAt             time.Time `json:"bookedAt"`
	BookingDate          string    `json:"bookingDate"`
	ValutaDate           string    `json:"valutaDate"`
	Status               string    `json:"status"`
	Asset                string    `json:"asset"`
	AmountInMinors       int64     `json:"amountInMinors"`
	BankTransactionCode  string    `json:"bankTransactionCode"`
	IsReversal           bool      `json:"isReversal"`
	IsBatch              bool      `json:"isBatch"`
	NumberOfTransactions uint64    `json:"numberOfTransactions"`

	EntryReference     string `json:"entryReference,omitempty"`
	ServicerReference  string `json:"servicerReference,omitempty"`
	BatchMessageID     string `json:"batchMessageId,omitempty"`
	BatchPaymentInfoID string `json:"batchPaymentInfoId,omitempty"`

	// There might be fewer details than there are transactions in bulk transactions
	// the bank does not always provide a detailed breakdown of all the underlying transactions
	Details []json.RawMessage `json:"details"`

	ImportedAt time.Time `json:"importedAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (c *client) GetTransactions(ctx context.Context, cursor string, pageSize int) ([]Transaction, bool, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	endpoint := fmt.Sprintf("%s/v1/connectors/transactions", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, false, "", fmt.Errorf("failed to create transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("pageSize", strconv.Itoa(pageSize))
	q.Add("cursor", cursor)
	req.URL.RawQuery = q.Encode()

	var body struct {
		Cursor struct {
			PageSize int64         `json:"pageSize"`
			Next     string        `json:"next"`
			Previous string        `json:"previous"`
			HasMore  bool          `json:"hasMore"`
			Data     []Transaction `json:"data"`
		} `json:"cursor"`
	}
	statusCode, err := c.httpClient.Do(ctx, req, &body, nil)
	if err != nil {
		return nil, false, "", errors.NewWrappedError(
			fmt.Errorf("failed to get transactions, status code: %d", statusCode),
			err,
		)
	}
	return body.Cursor.Data, body.Cursor.HasMore, body.Cursor.Next, nil
}
