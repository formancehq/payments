package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/go-libs/v2/errorsutils"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
)

type Refund struct {
	ID                     string `json:"Id"`
	Tag                    string `json:"Tag"`
	CreationDate           int64  `json:"CreationDate"`
	AuthorId               string `json:"AuthorId"`
	CreditedUserId         string `json:"CreditedUserId"`
	DebitedFunds           Funds  `json:"DebitedFunds"`
	CreditedFunds          Funds  `json:"CreditedFunds"`
	Fees                   Funds  `json:"Fees"`
	Status                 string `json:"Status"`
	ResultCode             string `json:"ResultCode"`
	ResultMessage          string `json:"ResultMessage"`
	ExecutionDate          int64  `json:"ExecutionDate"`
	Type                   string `json:"Type"`
	DebitedWalletId        string `json:"DebitedWalletId"`
	CreditedWalletId       string `json:"CreditedWalletId"`
	InitialTransactionID   string `json:"InitialTransactionId"`
	InitialTransactionType string `json:"InitialTransactionType"`
}

func (c *client) GetRefund(ctx context.Context, refundID string) (*Refund, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "get_refund")

	endpoint := fmt.Sprintf("%s/v2.01/%s/refunds/%s", c.endpoint, c.clientID, refundID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get refund request: %w", err)
	}

	var refund Refund
	statusCode, err := c.httpClient.Do(ctx, req, &refund, nil)
	if err != nil {
		return nil, errorsutils.NewErrorWithExitCode(fmt.Errorf("failed to get refund: %w", err), statusCode)
	}
	return &refund, nil
}
