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
	AccountID           string `json:"account_id"`
	Amount              int64  `json:"amount"`
	IndividualName      string `json:"individual_name"`
	ExternalAccountID   string `json:"external_account_id"`
	StatementDescriptor string `json:"statement_descriptor"`
}

type ACHPayoutResponse struct {
	ID                   string `json:"id"`
	TransactionID        string `json:"transaction_id"`
	PendingTransactionID string `json:"pending_transaction_id"`
	AccountID            string `json:"account_id"`
	AccountNumber        string `json:"account_number"`
	Amount               int64  `json:"amount"`
	Currency             string `json:"currency"`
	CreatedAt            string `json:"created_at"`
	Status               string `json:"status"`
	ExternalAccountID    string `json:"external_account_id"`
}

type WireTransferPayoutRequest struct {
	AccountID          string `json:"account_id"`
	Amount             int64  `json:"amount"`
	ExternalAccountID  string `json:"external_account_id"`
	BeneficiaryName    string `json:"beneficiary_name"`
	MessageToRecipient string `json:"message_to_recipient"`
}

type WireTransferPayoutResponse struct {
	ID                   string `json:"id"`
	TransactionID        string `json:"transaction_id"`
	PendingTransactionID string `json:"pending_transaction_id"`
	AccountID            string `json:"account_id"`
	AccountNumber        string `json:"account_number"`
	Amount               int64  `json:"amount"`
	Currency             string `json:"currency"`
	BeneficiaryName      string `json:"beneficiary_name"`
	CreatedAt            string `json:"created_at"`
	Status               string `json:"status"`
	RoutingNumber        string `json:"routing_number"`
	ExternalAccountID    string `json:"external_account_id"`
}

type MailingAddress struct {
	City       string  `json:"city"`
	Line1      string  `json:"line1"`
	Line2      *string `json:"line2,omitempty"`
	Name       *string `json:"name,omitempty"`
	PostalCode string  `json:"postal_code"`
	State      string  `json:"state"`
}

type PhysicalCheck struct {
	MailingAddress MailingAddress `json:"mailing_address"`
	Memo           string         `json:"memo"`
	RecipientName  string         `json:"recipient_name"`
}

type ThirdParty struct {
	CheckNumber string `json:"check_number"`
}

type CheckPayoutRequest struct {
	AccountID             string         `json:"account_id"`
	Amount                int64          `json:"amount"`
	SourceAccountNumberID string         `json:"source_account_number_id"`
	FulfillmentMethod     string         `json:"fulfillment_method"`
	PhysicalCheck         *PhysicalCheck `json:"physical_check,omitempty"`
	ThirdParty            *ThirdParty    `json:"third_party,omitempty"`
}

type CheckPayoutResponse struct {
	ID                            string         `json:"id"`
	TransactionID                 string         `json:"transaction_id"`
	AccountID                     string         `json:"account_id"`
	AccountNumber                 string         `json:"account_number"`
	Amount                        int64          `json:"amount"`
	Currency                      string         `json:"currency"`
	CheckNumber                   string         `json:"check_number"`
	CreatedAt                     string         `json:"created_at"`
	Status                        string         `json:"status"`
	RoutingNumber                 string         `json:"routing_number"`
	ApprovedInboundCheckDepositID string         `json:"approved_inbound_check_deposit_id"`
	FulfillmentMethod             string         `json:"fulfillment_method"`
	PendingTransactionID          string         `json:"pending_transaction_id"`
	PhysicalCheck                 *PhysicalCheck `json:"physical_check,omitempty"`
	SourceAccountNumberID         string         `json:"source_account_number_id"`
	ThirdParty                    *ThirdParty    `json:"third_party,omitempty"`
	Type                          string         `json:"type"`
}

type RTPPayoutRequest struct {
	Amount                int64  `json:"amount"`
	CreditorName          string `json:"creditor_name"`
	ExternalAccountID     string `json:"external_account_id"`
	SourceAccountNumberID string `json:"source_account_number_id"`
	RemittanceInformation string `json:"remittance_information"`
}

type RTPPayoutResponse struct {
	ID                       string `json:"id"`
	TransactionID            string `json:"transaction_id"`
	PendingTransactionID     string `json:"pending_transaction_id"`
	AccountID                string `json:"account_id"`
	Amount                   int64  `json:"amount"`
	Currency                 string `json:"currency"`
	CreatedAt                string `json:"created_at"`
	Status                   string `json:"status"`
	CreditorName             string `json:"creditor_name"`
	DestinationAccountNumber string `json:"destination_account_number"`
	DestinationRoutingNumber string `json:"destination_routing_number"`
	ExternalAccountID        string `json:"external_account_id"`
}

type PayoutResponse struct {
	ID                   string `json:"id"`
	TransactionID        string `json:"transaction_id"`
	PendingTransactionID string `json:"pending_transaction_id"`
	AccountID            string `json:"account_id"`
	Amount               int64  `json:"amount"`
	Currency             string `json:"currency"`
	CreatedAt            string `json:"created_at"`
	Status               string `json:"status"`
	RecipientName        string `json:"recipient_name"`
	ExternalAccountId    string `json:"external_account_id"`
	AccountNumber        string `json:"account_number"`
	RoutingNumber        string `json:"routing_number"`
	CheckNumber          string `json:"check_number"`
}

func (c *client) InitiateACHTransferPayout(ctx context.Context, pr *ACHPayoutRequest, idempotencyKey string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_ach_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "ach_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create ach payout request: %w", err)
	}
	req.Header.Add("Idempotency-Key", idempotencyKey)

	var res ACHPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create ach payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                   res.ID,
		TransactionID:        res.TransactionID,
		PendingTransactionID: res.PendingTransactionID,
		AccountID:            res.AccountID,
		Amount:               res.Amount,
		Currency:             res.Currency,
		CreatedAt:            res.CreatedAt,
		Status:               res.Status,
		ExternalAccountId:    res.ExternalAccountID,
		AccountNumber:        res.AccountNumber,
	}, nil
}

func (c *client) InitiateWireTransferPayout(ctx context.Context, pr *WireTransferPayoutRequest, idempotencyKey string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_wire_transfer_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "wire_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create wire transfer payout request: %w", err)
	}
	req.Header.Add("Idempotency-Key", idempotencyKey)

	var res WireTransferPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create wire transfer payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                   res.ID,
		TransactionID:        res.TransactionID,
		PendingTransactionID: res.PendingTransactionID,
		AccountID:            res.AccountID,
		Amount:               res.Amount,
		Currency:             res.Currency,
		CreatedAt:            res.CreatedAt,
		Status:               res.Status,
		ExternalAccountId:    res.ExternalAccountID,
		AccountNumber:        res.AccountNumber,
		RecipientName:        res.BeneficiaryName,
		RoutingNumber:        res.RoutingNumber,
	}, nil
}

func (c *client) InitiateCheckTransferPayout(ctx context.Context, pr *CheckPayoutRequest, idempotencyKey string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_check_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "check_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create check transfer payout request: %w", err)
	}
	req.Header.Add("Idempotency-Key", idempotencyKey)

	var res CheckPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create check transfer payout: %w %w", err, errRes.Error())
	}

	payoutResponse := &PayoutResponse{
		ID:                   res.ID,
		TransactionID:        res.TransactionID,
		PendingTransactionID: res.PendingTransactionID,
		AccountID:            res.AccountID,
		Amount:               res.Amount,
		Currency:             res.Currency,
		CreatedAt:            res.CreatedAt,
		Status:               res.Status,
		AccountNumber:        res.AccountNumber,
		RoutingNumber:        res.RoutingNumber,
		CheckNumber:          res.CheckNumber,
		// check transfer third party has no external account id and recipient name.
		// setting it to empty throws an unmarshal error in the engine
		RecipientName:     "Unknown",
		ExternalAccountId: "Unknown",
	}

	if res.PhysicalCheck != nil {
		payoutResponse.RecipientName = res.PhysicalCheck.RecipientName
	}

	return payoutResponse, nil
}

func (c *client) InitiateRTPTransferPayout(ctx context.Context, pr *RTPPayoutRequest, idempotencyKey string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_rtp_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "real_time_payments_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create real time payments transfer payout request: %w", err)
	}
	req.Header.Add("Idempotency-Key", idempotencyKey)

	var res RTPPayoutResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create real time payments transfer payout: %w %w", err, errRes.Error())
	}

	return &PayoutResponse{
		ID:                   res.ID,
		TransactionID:        res.TransactionID,
		PendingTransactionID: res.PendingTransactionID,
		AccountID:            res.AccountID,
		Amount:               res.Amount,
		Currency:             res.Currency,
		CreatedAt:            res.CreatedAt,
		Status:               res.Status,
		ExternalAccountId:    res.ExternalAccountID,
		AccountNumber:        res.DestinationAccountNumber,
		RoutingNumber:        res.DestinationRoutingNumber,
		RecipientName:        res.CreditorName,
	}, nil
}
