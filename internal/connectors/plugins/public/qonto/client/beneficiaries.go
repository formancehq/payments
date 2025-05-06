package client

import (
	"context"
	"fmt"
	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"net/http"
)

type BeneficiaryBankAccount struct {
	Iban                string `json:"iban"`
	Bic                 string `json:"bic"`
	Currency            string `json:"currency"`
	AccountNUmber       string `json:"account_number"`
	RoutingNumber       string `json:"routing_number"`
	IntermediaryBankBic string `json:"intermediary_bank_bic"`
	SwiftSortCode       string `json:"swift_sort_code"`
}

type Beneficiary struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	Trusted     bool                   `json:"trusted"`
	BankAccount BeneficiaryBankAccount `json:"bank_account"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

func (c *client) GetBeneficiaries(ctx context.Context, page, pageSize int) ([]Beneficiary, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("v2/beneficiaries"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("page", fmt.Sprint(page))
	q.Add("per_page", fmt.Sprint(pageSize))
	q.Add("sort_by", "updated_at:asc")
	// TODO the API supports a "updated_at_from" that we could make use of (we'd probably have to change the way we handle pages though)
	req.URL.RawQuery = q.Encode()

	errorResponse := qontoErrors{}
	type qontoResponse struct {
		Beneficiaries []Beneficiary  `json:"beneficiaries"`
		Meta          MetaPagination `json:"meta"`
	}
	successResponse := qontoResponse{}

	_, err = c.httpClient.Do(ctx, req, &successResponse, &errorResponse)

	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get beneficiaries: %v", errorResponse.Error()),
			err,
		)
	}
	return successResponse.Beneficiaries, nil
}
