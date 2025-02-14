package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type ACHPayoutRequest struct {
	AccountID           string      `json:"account_id"`
	Amount              json.Number `json:"amount"`
	IndividualName      string      `json:"individual_name"`
	ExternalAccountID   string      `json:"external_account_id"`
	StatementDescriptor string      `json:"statement_descriptor"`
}

type ACHPayoutResponse struct {
	ID                string      `json:"id"`
	AccountID         string      `json:"account_id"`
	AccountNumber     string      `json:"account_number"`
	Amount            json.Number `json:"amount"`
	Currency          string      `json:"currency"`
	CreatedAt         string      `json:"created_at"`
	Status            string      `json:"status"`
	ExternalAccountID string      `json:"external_account_id"`
}

type WireTransferPayoutRequest struct {
	AccountID          string      `json:"account_id"`
	Amount             json.Number `json:"amount"`
	ExternalAccountID  string      `json:"external_account_id"`
	BeneficiaryName    string      `json:"beneficiary_name"`
	MessageToRecipient string      `json:"message_to_recipient"`
}

type WireTransferPayoutResponse struct {
	ID                string      `json:"id"`
	AccountID         string      `json:"account_id"`
	AccountNumber     string      `json:"account_number"`
	Amount            json.Number `json:"amount"`
	Currency          string      `json:"currency"`
	BeneficiaryName   string      `json:"beneficiary_name"`
	CreatedAt         string      `json:"created_at"`
	Status            string      `json:"status"`
	RoutingNumber     string      `json:"routing_number"`
	ExternalAccountID string      `json:"external_account_id"`
}

type MailingAddress struct {
	City       string `json:"city"`
	Line1      string `json:"line1"`
	PostalCode string `json:"postal_code"`
	State      string `json:"state"`
}

type PhysicalCheck struct {
	MailingAddress MailingAddress `json:"mailing_address"`
	Memo           string         `json:"memo"`
	RecipientName  string         `json:"recipient_name"`
}

type CheckPayoutRequest struct {
	AccountID             string        `json:"account_id"`
	Amount                json.Number   `json:"amount"`
	SourceAccountNumberID string        `json:"source_account_number_id"`
	FulfillmentMethod     string        `json:"fulfillment_method"`
	PhysicalCheck         PhysicalCheck `json:"physical_check"`
	ThirdParty            struct {
		CheckNumber string `json:"check_number"`
	} `json:"third_party"`
}

type CheckPayoutResponse struct {
	ID            string      `json:"id"`
	AccountID     string      `json:"account_id"`
	AccountNumber string      `json:"account_number"`
	Amount        json.Number `json:"amount"`
	Currency      string      `json:"currency"`
	CheckNumber   string      `json:"check_number"`
	CreatedAt     string      `json:"created_at"`
	Status        string      `json:"status"`
	RoutingNumber string      `json:"routing_number"`
}

type RTPPayoutRequest struct {
	Amount                json.Number `json:"amount"`
	CreditorName          string      `json:"creditor_name"`
	ExternalAccountID     string      `json:"external_account_id"`
	SourceAccountNumberID string      `json:"source_account_number_id"`
	RemittanceInformation string      `json:"remittance_information"`
}

type RTPPayoutResponse struct {
	ID                       string      `json:"id"`
	AccountID                string      `json:"account_id"`
	Amount                   json.Number `json:"amount"`
	Currency                 string      `json:"currency"`
	CreatedAt                string      `json:"created_at"`
	Status                   string      `json:"status"`
	CreditorName             string      `json:"creditor_name"`
	DestinationAccountNumber string      `json:"destination_account_number"`
	DestinationRoutingNumber string      `json:"destination_routing_number"`
	ExternalAccountID        string      `json:"external_account_id"`
}

type PayoutResponse struct {
	ID                string      `json:"id"`
	AccountID         string      `json:"account_id"`
	Amount            json.Number `json:"amount"`
	Currency          string      `json:"currency"`
	CreatedAt         string      `json:"created_at"`
	Status            string      `json:"status"`
	RecipientName     string      `json:"reciepient_name"`
	ExternalAccountId string      `json:"external_account_id"`
	AccountNumber     string      `json:"account_number"`
	RoutingNumber     string      `json:"routing_number"`
	CheckNumber       string      `json:"check_number"`
}

func (c *client) InitiateACHTransferPayout(ctx context.Context, pr *ACHPayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_ach_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "ach_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create ach payout request: %w", err)
	}

	var res ACHPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create ach payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                res.ID,
		AccountID:         res.AccountID,
		Amount:            res.Amount,
		Currency:          res.Currency,
		CreatedAt:         res.CreatedAt,
		Status:            res.Status,
		ExternalAccountId: res.ExternalAccountID,
		AccountNumber:     res.AccountNumber,
	}, nil
}

func (c *client) GetACHTransferPayout(ctx context.Context, transferID string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_ach_payout")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("ach_transfers/%s", transferID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create ach payout request: %w", err)
	}

	var res ACHPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get ach payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                res.ID,
		AccountID:         res.AccountID,
		Amount:            res.Amount,
		Currency:          res.Currency,
		CreatedAt:         res.CreatedAt,
		Status:            res.Status,
		ExternalAccountId: res.ExternalAccountID,
		AccountNumber:     res.AccountNumber,
	}, nil
}

func (c *client) InitiateWireTransferPayout(ctx context.Context, pr *WireTransferPayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_wire_transfer_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "wire_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create wire transfer payout request: %w", err)
	}

	var res WireTransferPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create wire transfer payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                res.ID,
		AccountID:         res.AccountID,
		Amount:            res.Amount,
		Currency:          res.Currency,
		CreatedAt:         res.CreatedAt,
		Status:            res.Status,
		ExternalAccountId: res.ExternalAccountID,
		AccountNumber:     res.AccountNumber,
		RecipientName:     res.BeneficiaryName,
		RoutingNumber:     res.RoutingNumber,
	}, nil
}

func (c *client) GetWireTransferPayout(ctx context.Context, transferID string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_wire_transfer_payout")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("wire_transfers/%s", transferID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create wire transfer payout request: %w", err)
	}

	var res WireTransferPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get wire transfer payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                res.ID,
		AccountID:         res.AccountID,
		Amount:            res.Amount,
		Currency:          res.Currency,
		CreatedAt:         res.CreatedAt,
		Status:            res.Status,
		ExternalAccountId: res.ExternalAccountID,
		AccountNumber:     res.AccountNumber,
		RecipientName:     res.BeneficiaryName,
		RoutingNumber:     res.RoutingNumber,
	}, nil
}

func (c *client) InitiateCheckTransferPayout(ctx context.Context, pr *CheckPayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_check_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "check_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create check transfer payout request: %w", err)
	}

	var res CheckPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create check transfer payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:            res.ID,
		AccountID:     res.AccountID,
		Amount:        res.Amount,
		Currency:      res.Currency,
		CreatedAt:     res.CreatedAt,
		Status:        res.Status,
		AccountNumber: res.AccountNumber,
		RoutingNumber: res.RoutingNumber,
		CheckNumber:   res.CheckNumber,
	}, nil
}

func (c *client) GetCheckTransferPayout(ctx context.Context, transferID string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_check_payout")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("check_transfers/%s", transferID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create check transfer payout request: %w", err)
	}

	var res CheckPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get check transfer payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:            res.ID,
		AccountID:     res.AccountID,
		Amount:        res.Amount,
		Currency:      res.Currency,
		CreatedAt:     res.CreatedAt,
		Status:        res.Status,
		AccountNumber: res.AccountNumber,
		RoutingNumber: res.RoutingNumber,
		CheckNumber:   res.CheckNumber,
	}, nil
}

func (c *client) InitiateRTPTransferPayout(ctx context.Context, pr *RTPPayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_rtp_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "real_time_payments_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create real time payments transfer payout request: %w", err)
	}

	var res RTPPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create real time payments transfer payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                res.ID,
		AccountID:         res.AccountID,
		Amount:            res.Amount,
		Currency:          res.Currency,
		CreatedAt:         res.CreatedAt,
		Status:            res.Status,
		ExternalAccountId: res.ExternalAccountID,
		AccountNumber:     res.DestinationAccountNumber,
		RoutingNumber:     res.DestinationRoutingNumber,
		RecipientName:     res.CreditorName,
	}, nil
}

func (c *client) GetRTPTransferPayout(ctx context.Context, transferID string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_rtp_payout")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("real_time_payments_transfers/%s", transferID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create real time payments transfer payout request: %w", err)
	}

	var res RTPPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get real time payments transfer payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                res.ID,
		AccountID:         res.AccountID,
		Amount:            res.Amount,
		Currency:          res.Currency,
		CreatedAt:         res.CreatedAt,
		Status:            res.Status,
		ExternalAccountId: res.ExternalAccountID,
		AccountNumber:     res.DestinationAccountNumber,
		RoutingNumber:     res.DestinationRoutingNumber,
		RecipientName:     res.CreditorName,
	}, nil
}
