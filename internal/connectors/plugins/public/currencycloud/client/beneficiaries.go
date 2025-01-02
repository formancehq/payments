package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Beneficiary struct {
	ID                    string    `json:"id"`
	BankAccountHolderName string    `json:"bank_account_holder_name"`
	Name                  string    `json:"name"`
	Currency              string    `json:"currency"`
	CreatedAt             time.Time `json:"created_at"`
	// Contains a lot more fields that will be not used on our side for now
}

func (c *client) GetBeneficiaries(ctx context.Context, page int, pageSize int) ([]*Beneficiary, int, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_beneficiaries")

	if err := c.ensureLogin(ctx); err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.buildEndpoint("v2/beneficiaries/find"), http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("page", fmt.Sprint(page))
	q.Add("per_page", fmt.Sprint(pageSize))
	q.Add("order", "created_at")
	q.Add("order_asc_desc", "asc")
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Accept", "application/json")

	//nolint:tagliatelle // allow for client code
	type response struct {
		Beneficiaries []*Beneficiary `json:"beneficiaries"`
		Pagination    struct {
			NextPage int `json:"next_page"`
		} `json:"pagination"`
	}

	res := response{Beneficiaries: make([]*Beneficiary, 0)}
	var errRes currencyCloudError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get beneficiaries %w, %w", err, errRes.Error())
	}
	return res.Beneficiaries, res.Pagination.NextPage, nil
}
