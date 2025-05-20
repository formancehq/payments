package client

import (
	"context"
	"fmt"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type OrganizationBankAccount struct {
	Id                     string  `json:"id"`
	Slug                   string  `json:"slug"`
	Iban                   string  `json:"iban"`
	Bic                    string  `json:"bic"`
	Currency               string  `json:"currency"`
	Balance                float64 `json:"balance"` // TODO change to JSON.Number and use currency function
	BalanceCents           int64   `json:"balance_cents"`
	AuthorizedBalance      float64 `json:"authorized_balance"`
	AuthorizedBalanceCents int64   `json:"authorized_balance_cents"`
	Name                   string  `json:"name"`
	UpdatedAt              string  `json:"updated_at"`
	Status                 string  `json:"status"`
	Main                   bool    `json:"main"`
	IsExternalAccount      bool    `json:"is_external_account"`
	AccountNumber          string  `json:"account_number,omitempty"`
}

type Organization struct {
	Id                    string                    `json:"id"`
	Name                  string                    `json:"name"`
	Slug                  string                    `json:"slug"`
	LegalName             string                    `json:"legal_name,omitempty"`
	Locale                string                    `json:"locale"`
	LegalShareCapital     float64                   `json:"legal_share_capital"`
	LegalCountry          string                    `json:"legal_country"`
	LegalRegistrationDate string                    `json:"legal_registration_date,omitempty"`
	LegalForm             string                    `json:"legal_form"`
	LegalAddress          string                    `json:"legal_address"`
	LegalSector           string                    `json:"legal_sector,omitempty"`
	ContractSignedAt      string                    `json:"contract_signed_at"`
	LegalNumber           string                    `json:"legal_number"`
	BankAccounts          []OrganizationBankAccount `json:"bank_accounts"`
}

func (c *client) GetOrganization(ctx context.Context) (*Organization, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("v2/organization"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	errorResponse := qontoErrors{}
	type qontoResponse struct {
		Organization Organization `json:"organization"`
	}
	successResponse := qontoResponse{}

	_, err = c.httpClient.Do(ctx, req, &successResponse, &errorResponse)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get organization: %v", errorResponse.Error()),
			err,
		)
	}
	return &successResponse.Organization, nil
}
