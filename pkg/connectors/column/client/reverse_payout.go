package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type ReversePayoutReason string

const (
	ReasonDuplicatedEntry          ReversePayoutReason = "duplicated_entry"
	ReasonIncorrectAmount          ReversePayoutReason = "incorrect_amount"
	ReasonIncorrectReceiverAccount ReversePayoutReason = "incorrect_receiver_account"
	ReasonDebitEarlierThanIntended ReversePayoutReason = "debit_earlier_than_intended"
	ReasonCreditLaterThanIntended  ReversePayoutReason = "credit_later_than_intended"
)

type ReversePayoutRequest struct {
	Reason        ReversePayoutReason `json:"reason"`
	ACHTransferID string              `json:"ach_transfer_id"`
	Description   string              `json:"description,omitempty"`
}

type ReversePayoutResponse struct {
	ID                                string  `json:"id"`
	CreatedAt                         string  `json:"created_at"`
	UpdatedAt                         string  `json:"updated_at"`
	InitiatedAt                       string  `json:"initiated_at"`
	CompletedAt                       *string `json:"completed_at"`
	AcknowledgedAt                    *string `json:"acknowledged_at"`
	CancelledAt                       *string `json:"cancelled_at"`
	ManualReviewAt                    *string `json:"manual_review_at"`
	ReturnContestedAt                 *string `json:"return_contested_at"`
	ReturnDishonoredAt                *string `json:"return_dishonored_at"`
	ReturnedAt                        *string `json:"returned_at"`
	SettledAt                         *string `json:"settled_at"`
	SubmittedAt                       *string `json:"submitted_at"`
	EffectiveOn                       string  `json:"effective_on"`
	NSFDeadline                       *string `json:"nsf_deadline"`
	IdempotencyKey                    string  `json:"idempotency_key"`
	AccountNumberID                   string  `json:"account_number_id"`
	BankAccountID                     string  `json:"bank_account_id"`
	CounterpartyID                    string  `json:"counterparty_id"`
	Amount                            int64   `json:"amount"`
	CurrencyCode                      string  `json:"currency_code"`
	Description                       string  `json:"description"`
	Status                            string  `json:"status"`
	Type                              string  `json:"type"`
	CompanyID                         string  `json:"company_id"`
	CompanyName                       string  `json:"company_name"`
	CompanyEntryDescription           string  `json:"company_entry_description"`
	CompanyDiscretionaryData          string  `json:"company_discretionary_data"`
	EntryClassCode                    string  `json:"entry_class_code"`
	ODFIRoutingNumber                 string  `json:"odfi_routing_number"`
	ReceiverID                        string  `json:"receiver_id"`
	ReceiverName                      string  `json:"receiver_name"`
	TraceNumber                       string  `json:"trace_number"`
	PaymentRelatedInfo                string  `json:"payment_related_info"`
	ReversalPairTransferID            string  `json:"reversal_pair_transfer_id"`
	UltimateBeneficiaryCounterpartyID string  `json:"ultimate_beneficiary_counterparty_id"`
	UltimateOriginatorCounterpartyID  string  `json:"ultimate_originator_counterparty_id"`
	AllowOverdraft                    bool    `json:"allow_overdraft"`
	IsIncoming                        bool    `json:"is_incoming"`
	IsOnUs                            bool    `json:"is_on_us"`
	SameDay                           bool    `json:"same_day"`
	IAT                               string  `json:"iat"`
	ReturnDetails                     []any   `json:"return_details"`
	NotificationOfChanges             string  `json:"notification_of_changes"`
}

func (c *client) ReversePayout(ctx context.Context, req *ReversePayoutRequest) (*ReversePayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "reverse_payout")

	body, err := json.Marshal(req)
	if err != nil {
		return &ReversePayoutResponse{}, err
	}

	endpoint := fmt.Sprintf("transfers/ach/%s/reverse", req.ACHTransferID)

	request, err := c.newRequest(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return &ReversePayoutResponse{}, fmt.Errorf("failed to create reverse payout request: %w", err)
	}

	var response ReversePayoutResponse
	var errRes columnError
	_, err = c.httpClient.Do(ctx, request, &response, &errRes)
	if err != nil {
		return &ReversePayoutResponse{}, fmt.Errorf("failed to send reverse payout request: %w", err)
	}

	return &response, nil
}
