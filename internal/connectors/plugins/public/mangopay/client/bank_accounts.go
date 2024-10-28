package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/go-libs/v2/errorsutils"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
)

type OwnerAddress struct {
	AddressLine1 string `json:"AddressLine1,omitempty"`
	AddressLine2 string `json:"AddressLine2,omitempty"`
	City         string `json:"City,omitempty"`
	// Region is needed if country is either US, CA or MX
	Region     string `json:"Region,omitempty"`
	PostalCode string `json:"PostalCode,omitempty"`
	// ISO 3166-1 alpha-2 format.
	Country string `json:"Country,omitempty"`
}

type CreateIBANBankAccountRequest struct {
	OwnerName    string        `json:"OwnerName"`
	OwnerAddress *OwnerAddress `json:"OwnerAddress,omitempty"`
	IBAN         string        `json:"IBAN,omitempty"`
	BIC          string        `json:"BIC,omitempty"`
	// Metadata
	Tag string `json:"Tag,omitempty"`
}

func (c *client) CreateIBANBankAccount(ctx context.Context, userID string, req *CreateIBANBankAccountRequest) (*BankAccount, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "create_iban_bank_account")

	endpoint := fmt.Sprintf("%s/v2.01/%s/users/%s/bankaccounts/iban", c.endpoint, c.clientID, userID)
	return c.createBankAccount(ctx, endpoint, req)
}

type CreateUSBankAccountRequest struct {
	OwnerName          string        `json:"OwnerName"`
	OwnerAddress       *OwnerAddress `json:"OwnerAddress,omitempty"`
	AccountNumber      string        `json:"AccountNumber"`
	ABA                string        `json:"ABA"`
	DepositAccountType string        `json:"DepositAccountType,omitempty"`
	Tag                string        `json:"Tag,omitempty"`
}

func (c *client) CreateUSBankAccount(ctx context.Context, userID string, req *CreateUSBankAccountRequest) (*BankAccount, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "create_us_bank_account")

	endpoint := fmt.Sprintf("%s/v2.01/%s/users/%s/bankaccounts/us", c.endpoint, c.clientID, userID)
	return c.createBankAccount(ctx, endpoint, req)
}

type CreateCABankAccountRequest struct {
	OwnerName         string        `json:"OwnerName"`
	OwnerAddress      *OwnerAddress `json:"OwnerAddress,omitempty"`
	AccountNumber     string        `json:"AccountNumber"`
	InstitutionNumber string        `json:"InstitutionNumber"`
	BranchCode        string        `json:"BranchCode"`
	BankName          string        `json:"BankName"`
	Tag               string        `json:"Tag,omitempty"`
}

func (c *client) CreateCABankAccount(ctx context.Context, userID string, req *CreateCABankAccountRequest) (*BankAccount, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "create_ca_bank_account")

	endpoint := fmt.Sprintf("%s/v2.01/%s/users/%s/bankaccounts/ca", c.endpoint, c.clientID, userID)
	return c.createBankAccount(ctx, endpoint, req)
}

type CreateGBBankAccountRequest struct {
	OwnerName     string        `json:"OwnerName"`
	OwnerAddress  *OwnerAddress `json:"OwnerAddress,omitempty"`
	AccountNumber string        `json:"AccountNumber"`
	SortCode      string        `json:"SortCode"`
	Tag           string        `json:"Tag,omitempty"`
}

func (c *client) CreateGBBankAccount(ctx context.Context, userID string, req *CreateGBBankAccountRequest) (*BankAccount, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "create_gb_bank_account")

	endpoint := fmt.Sprintf("%s/v2.01/%s/users/%s/bankaccounts/gb", c.endpoint, c.clientID, userID)
	return c.createBankAccount(ctx, endpoint, req)
}

type CreateOtherBankAccountRequest struct {
	OwnerName     string        `json:"OwnerName"`
	OwnerAddress  *OwnerAddress `json:"OwnerAddress,omitempty"`
	AccountNumber string        `json:"AccountNumber"`
	BIC           string        `json:"BIC,omitempty"`
	Country       string        `json:"Country,omitempty"`
	Tag           string        `json:"Tag,omitempty"`
}

func (c *client) CreateOtherBankAccount(ctx context.Context, userID string, req *CreateOtherBankAccountRequest) (*BankAccount, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "create_other_bank_account")

	endpoint := fmt.Sprintf("%s/v2.01/%s/users/%s/bankaccounts/other", c.endpoint, c.clientID, userID)
	return c.createBankAccount(ctx, endpoint, req)
}

func (c *client) createBankAccount(ctx context.Context, endpoint string, req any) (*BankAccount, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bank account request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create bank account request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	var bankAccount BankAccount
	statusCode, err := c.httpClient.Do(ctx, httpReq, &bankAccount, nil)
	if err != nil {
		return nil, errorsutils.NewErrorWithExitCode(fmt.Errorf("failed to create bank account: %w", err), statusCode)
	}
	return &bankAccount, nil
}

type BankAccount struct {
	ID           string `json:"Id"`
	OwnerName    string `json:"OwnerName"`
	CreationDate int64  `json:"CreationDate"`
}

func (c *client) GetBankAccounts(ctx context.Context, userID string, page, pageSize int) ([]BankAccount, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "list_bank_accounts")

	endpoint := fmt.Sprintf("%s/v2.01/%s/users/%s/bankaccounts", c.endpoint, c.clientID, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	q := req.URL.Query()
	q.Add("per_page", strconv.Itoa(pageSize))
	q.Add("page", fmt.Sprint(page))
	q.Add("Sort", "CreationDate:ASC")
	req.URL.RawQuery = q.Encode()

	var bankAccounts []BankAccount
	statusCode, err := c.httpClient.Do(ctx, req, &bankAccounts, nil)
	if err != nil {
		return nil, errorsutils.NewErrorWithExitCode(fmt.Errorf("failed to get bank accounts: %w", err), statusCode)
	}
	return bankAccounts, nil
}
