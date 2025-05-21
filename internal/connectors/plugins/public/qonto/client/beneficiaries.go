package client

import (
	"context"
	"fmt"
	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"net/http"
	"time"
)

type BeneficiaryBankAccount struct {
	Iban                string `json:"iban"`
	Bic                 string `json:"bic"`
	Currency            string `json:"currency"`
	AccountNumber       string `json:"account_number"`
	RoutingNumber       string `json:"routing_number"`
	IntermediaryBankBic string `json:"intermediary_bank_bic"`
	SwiftSortCode       string `json:"swift_sort_code"`
}

type Beneficiary struct {
	Id          string                 `json:"id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	Trusted     bool                   `json:"trusted"`
	BankAccount BeneficiaryBankAccount `json:"bank_account"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

func (c *client) GetBeneficiaries(ctx context.Context, updatedAtFrom time.Time, page, pageSize int) ([]Beneficiary, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("v2/beneficiaries"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	if !updatedAtFrom.IsZero() {
		// Qonto doesn't accept udpated_at_from too much in the past
  q.Add("updated_at_from", updatedAtFrom.Format(QontoTimeformat))
	}
	q.Add("per_page", fmt.Sprint(pageSize))
	q.Add("page", fmt.Sprint(page))
	q.Add("sort_by", "updated_at:asc")
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
			fmt.Errorf("failed to get beneficiaries: %w", errorResponse.Error()),
			err,
		)
	}
	if len(errorResponse.Errors) != 0 {
		return nil, fmt.Errorf("failed to get beneficiaries: %w", errorResponse.Error())
	}
	return successResponse.Beneficiaries, nil
}
