package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type CounterPartyBankAccountRequest struct {
	Name                string        `json:"name"`
	RoutingNumber       string        `json:"routing_number"`
	AccountNumber       string        `json:"account_number"`
	RoutingNumberType   string        `json:"routing_number_type,omitempty"`
	AccountType         string        `json:"account_type,omitempty"`
	WireDrawdownAllowed bool          `json:"wire_drawdown_allowed,omitempty"`
	Address             ColumnAddress `json:"address,omitempty"`
	Phone               string        `json:"phone,omitempty"`
	Email               string        `json:"email,omitempty"`
	LegalID             string        `json:"legal_id,omitempty"`
	LegalType           string        `json:"legal_type,omitempty"`
	LocalBankCode       string        `json:"local_bank_code,omitempty"`
	LocalAccountNumber  string        `json:"local_account_number,omitempty"`
}

type CounterPartyBankAccountResponse struct {
	AccountNumber        string  `json:"account_number"`
	AccountType          string  `json:"account_type"`
	Address              Address `json:"address"`
	CreatedAt            string  `json:"created_at"`
	Description          string  `json:"description"`
	Email                string  `json:"email"`
	ID                   string  `json:"id"`
	IsColumnAccount      bool    `json:"is_column_account"`
	LegalID              string  `json:"legal_id"`
	LegalType            string  `json:"legal_type"`
	LocalAccountNumber   string  `json:"local_account_number"`
	LocalBankCode        string  `json:"local_bank_code"`
	LocalBankCountryCode string  `json:"local_bank_country_code"`
	LocalBankName        string  `json:"local_bank_name"`
	Name                 string  `json:"name"`
	Phone                string  `json:"phone"`
	RoutingNumber        string  `json:"routing_number"`
	RoutingNumberType    string  `json:"routing_number_type"`
	UpdatedAt            string  `json:"updated_at"`
	Wire                 Wire    `json:"wire"`
	WireDrawdownAllowed  bool    `json:"wire_drawdown_allowed"`
}

type Address struct {
	City        string `json:"city"`
	CountryCode string `json:"country_code"`
	Line1       string `json:"line_1"`
	Line2       string `json:"line_2"`
	PostalCode  string `json:"postal_code"`
	State       string `json:"state"`
}

type Wire struct {
	BeneficiaryAddress Address `json:"beneficiary_address"`
	BeneficiaryEmail   string  `json:"beneficiary_email"`
	BeneficiaryLegalID string  `json:"beneficiary_legal_id"`
	BeneficiaryName    string  `json:"beneficiary_name"`
	BeneficiaryPhone   string  `json:"beneficiary_phone"`
	BeneficiaryType    string  `json:"beneficiary_type"`
	LocalAccountNumber string  `json:"local_account_number"`
	LocalBankCode      string  `json:"local_bank_code"`
}

func (c *client) CreateCounterPartyBankAccount(ctx context.Context, data CounterPartyBankAccountRequest) (CounterPartyBankAccountResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_external_bank_account")

	body, err := json.Marshal(data)
	if err != nil {
		return CounterPartyBankAccountResponse{}, fmt.Errorf("failed to marshal bank account request: %w", err)
	}
	req, err := c.newRequest(ctx, http.MethodPost, "counterparties", bytes.NewBuffer(body))
	if err != nil {
		return CounterPartyBankAccountResponse{}, fmt.Errorf("failed to create counter party bank account request: %w", err)
	}

	var response CounterPartyBankAccountResponse
	var errRes columnError
	if _, err := c.httpClient.Do(ctx, req, &response, &errRes); err != nil {
		return CounterPartyBankAccountResponse{}, fmt.Errorf("failed to create counter party bank account: %w %w", err, errRes.Error())
	}

	return response, nil
}
