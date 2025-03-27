package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type PayinResponse struct {
	ID               string `json:"Id"`
	Tag              string `json:"Tag"`
	CreationDate     int64  `json:"CreationDate"`
	ResultCode       string `json:"ResultCode"`
	ResultMessage    string `json:"ResultMessage"`
	AuthorId         string `json:"AuthorId"`
	CreditedUserId   string `json:"CreditedUserId"`
	DebitedFunds     Funds  `json:"DebitedFunds"`
	CreditedFunds    Funds  `json:"CreditedFunds"`
	Fees             Funds  `json:"Fees"`
	Status           string `json:"Status"`
	ExecutionDate    int64  `json:"ExecutionDate"`
	Type             string `json:"Type"`
	CreditedWalletID string `json:"CreditedWalletId"`
	PaymentType      string `json:"PaymentType"`
	ExecutionType    string `json:"ExecutionType"`
}

func (c *client) GetPayin(ctx context.Context, payinID string) (*PayinResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_payin")

	endpoint := fmt.Sprintf("%s/v2.01/%s/payins/%s", c.endpoint, c.clientID, payinID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get payin request: %w", err)
	}

	var payinResponse PayinResponse
	statusCode, err := c.httpClient.Do(ctx, req, &payinResponse, nil)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get payin: status code %d", statusCode),
			err,
		)
	}
	return &payinResponse, nil
}
