package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
)

type ACHPayoutRequest struct {
	AchCompanyID    string `json:"ach_company_id"`
	Amount          int64  `json:"amount,omitempty"`
	AmountCondition string `json:"amount_condition"`
	BankAccountID   string `json:"bank_account_id"`
	Description     string `json:"description"`
}

type ACHPayoutResponse struct {
	ACHCompanyID         string `json:"ach_company_id"`
	ACHPositivePayRuleID string `json:"ach_positive_pay_rule_id"`
	Amount               int64  `json:"amount,omitempty"`
	AmountCondition      string `json:"amount_condition"`
	BankAccountID        string `json:"bank_account_id"`
	Description          string `json:"description"`
}

type WirePayoutRequest struct {
	Amount                           int64  `json:"amount"`
	CurrencyCode                     string `json:"currency_code"`
	AccountNumberId                  string `json:"account_number_id"`
	BankAccountID                    string `json:"bank_account_id"`
	CounterPartyId                   string `json:"counterparty_id"`
	Description                      string `json:"description"`
	AllowOverdraft                   bool   `json:"allow_overdraft"`
	UltimateOriginatorCounterpartyId string `json:"ultimate_originator_counterparty_id"`
	UltimateOriginatorAccountNumber  string `json:"ultimate_originator_account_number"`
}

type WirePayoutResponse struct {
	ID                    string `json:"id"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
	InitiatedAt           string `json:"initiated_at"`
	PendingSubmissionAt   string `json:"pending_submission_at"`
	SubmittedAt           string `json:"submitted_at"`
	CompletedAt           string `json:"completed_at"`
	RejectedAt            string `json:"rejected_at"`
	IdempotencyKey        string `json:"idempotency_key"`
	BankAccountID         string `json:"bank_account_id"`
	AccountNumberID       string `json:"account_number_id"`
	CounterpartyID        string `json:"counterparty_id"`
	Amount                int64  `json:"amount"`
	CurrencyCode          string `json:"currency_code"`
	Description           string `json:"description"`
	Status                string `json:"status"`
	AllowOverdraft        bool   `json:"allow_overdraft"`
	PlatformID            string `json:"platform_id"`
	IsOnUs                bool   `json:"is_on_us"`
	IsIncoming            bool   `json:"is_incoming"`
	WireDrawdownRequestID string `json:"wire_drawdown_request_id"`
}

type InternationalWirePayoutRequest struct {
	AccountNumberId          string                       `json:"account_number_id,omitempty"`
	AllowOverdraft           bool                         `json:"allow_overdraft,omitempty"`
	Amount                   int64                        `json:"amount"`
	BankAccountID            string                       `json:"bank_account_id,omitempty"`
	ChargeBearer             string                       `json:"charge_bearer,omitempty"`
	CounterpartyID           string                       `json:"counterparty_id"`
	CurrencyCode             string                       `json:"currency_code"`
	Description              string                       `json:"description,omitempty"`
	FxQuoteID                string                       `json:"fx_quote_id,omitempty"`
	IntermediaryBank         string                       `json:"intermediary_bank,omitempty"`
	MessageToBeneficiaryBank string                       `json:"message_to_beneficiary_bank,omitempty"`
	RemittanceInformation    RemittanceInformationRequest `json:"remittance_information,omitempty"`
}

type RemittanceInformationRequest struct {
	GeneralInfo          string `json:"general_info,omitempty"`
	BeneficiaryReference string `json:"beneficiary_reference,omitempty"`
	PurposeCode          string `json:"purpose_code,omitempty"`
}

type Charge struct {
	Amount       int64  `json:"amount"`
	CurrencyCode string `json:"currency_code"`
	Agent        string `json:"agent"`
}

type InternationalWirePayoutResponse struct {
	ID                         string                       `json:"id"`
	CreatedAt                  string                       `json:"created_at"`
	UpdatedAt                  string                       `json:"updated_at"`
	InitiatedAt                string                       `json:"initiated_at"`
	PendingSubmissionAt        string                       `json:"pending_submission_at"`
	SubmittedAt                string                       `json:"submitted_at"`
	CompletedAt                *string                      `json:"completed_at"`
	ManualReviewAt             *string                      `json:"manual_review_at"`
	ReturnedAt                 *string                      `json:"returned_at"`
	IdempotencyKey             *string                      `json:"idempotency_key"`
	RawMessage                 *string                      `json:"raw_message"`
	ReturnReason               *string                      `json:"return_reason"`
	ReturnedAmount             *int64                       `json:"returned_amount"`
	ReturnedCurrencyCode       *string                      `json:"returned_currency_code"`
	SettlementDate             string                       `json:"settlement_date"`
	UETR                       string                       `json:"uetr"`
	InstructionID              string                       `json:"instruction_id"`
	EndToEndID                 string                       `json:"end_to_end_id"`
	Status                     string                       `json:"status"`
	AccountNumberID            string                       `json:"account_number_id"`
	BankAccountID              string                       `json:"bank_account_id"`
	CounterpartyID             string                       `json:"counterparty_id"`
	FXQuoteID                  string                       `json:"fx_quote_id"`
	FXRate                     string                       `json:"fx_rate"`
	Amount                     int64                        `json:"amount"`
	CurrencyCode               string                       `json:"currency_code"`
	InstructedAmount           int64                        `json:"instructed_amount"`
	InstructedCurrencyCode     string                       `json:"instructed_currency_code"`
	SettledAmount              int64                        `json:"settled_amount"`
	SettledCurrencyCode        string                       `json:"settled_currency_code"`
	AllowOverdraft             bool                         `json:"allow_overdraft"`
	IsIncoming                 bool                         `json:"is_incoming"`
	Description                string                       `json:"description"`
	InstructionToBeneficiaryFI string                       `json:"instruction_to_beneficiary_fi"`
	BeneficiaryAccountNumber   string                       `json:"beneficiary_account_number"`
	BeneficiaryFI              string                       `json:"beneficiary_fi"`
	BeneficiaryName            string                       `json:"beneficiary_name"`
	OriginatorAccountNumber    string                       `json:"originator_account_number"`
	OriginatorFI               string                       `json:"originator_fi"`
	OriginatorName             string                       `json:"originator_name"`
	UltimateBeneficiaryName    string                       `json:"ultimate_beneficiary_name"`
	UltimateOriginatorName     string                       `json:"ultimate_originator_name"`
	BeneficiaryAddress         Address                      `json:"beneficiary_address"`
	OriginatorAddress          Address                      `json:"originator_address"`
	UltimateBeneficiaryAddress *Address                     `json:"ultimate_beneficiary_address"`
	UltimateOriginatorAddress  *Address                     `json:"ultimate_originator_address"`
	ChargeBearer               string                       `json:"charge_bearer"`
	Charges                    []Charge                     `json:"charges"`
	RemittanceInfo             RemittanceInformationRequest `json:"remittance_info"`
}

type RealtimeTransferRequest struct {
	Amount                       int64  `json:"amount"`
	CurrencyCode                 string `json:"currency_code"`
	BankAccountID                string `json:"bank_account_id"`
	Description                  string `json:"description"`
	CounterpartyID               string `json:"counterparty_id"`
	AllowOverdraft               bool   `json:"allow_overdraft,omitempty"`
	AccountNumberId              string `json:"account_number_id,omitempty"`
	UltimateDebtorCounterparty   string `json:"ultimate_debtor_counterparty,omitempty"`
	UltimateDebtorCounterpartyID string `json:"ultimate_debtor_counterparty_id,omitempty"`
	EndToEndID                   string `json:"end_to_end_id,omitempty"`
}

type RealtimeTransferResponse struct {
	AcceptedAt                   *string `json:"accepted_at"`
	AccountNumberID              string  `json:"account_number_id"`
	AllowOverdraft               bool    `json:"allow_overdraft"`
	Amount                       int64   `json:"amount"`
	BankAccountID                string  `json:"bank_account_id"`
	BlockedAt                    *string `json:"blocked_at,omitempty"`
	CompletedAt                  *string `json:"completed_at"`
	CounterpartyID               string  `json:"counterparty_id"`
	UltimateDebtorCounterpartyID string  `json:"ultimate_debtor_counterparty_id,omitempty"`
	CurrencyCode                 string  `json:"currency_code"`
	Description                  string  `json:"description"`
	ID                           string  `json:"id"`
	IdempotencyKey               *string `json:"idempotency_key"`
	InitiatedAt                  string  `json:"initiated_at"`
	IsIncoming                   bool    `json:"is_incoming"`
	IsOnUs                       bool    `json:"is_on_us"`
	ManualReviewApprovedAt       *string `json:"manual_review_approved_at,omitempty"`
	ManualReviewAt               *string `json:"manual_review_at,omitempty"`
	ManualReviewRejectedAt       *string `json:"manual_review_rejected_at,omitempty"`
	PendingAt                    *string `json:"pending_at,omitempty"`
	RejectedAt                   *string `json:"rejected_at"`
	RejectedCode                 *string `json:"rejected_code,omitempty"`
	RejectionCodeDescription     *string `json:"rejection_code_description,omitempty"`
	RejectionAdditionalInfo      *string `json:"rejection_additional_info,omitempty"`
	Status                       string  `json:"status"`
	ReturnPairTransferID         *string `json:"return_pair_transfer_id,omitempty"`
}

type PayoutRequest struct {
	Amount             int64             `json:"amount"`
	CurrencyCode       string            `json:"currency_code"`
	SourceAccount      string            `json:"source_account"`
	DestinationAccount string            `json:"destination_account"`
	Description        string            `json:"description"`
	Metadata           map[string]string `json:"metadata"`
}

type PayoutResponse struct {
	ID                   string          `json:"id"`
	Amount               int64           `json:"amount"`
	AchPositivePayRuleID string          `json:"ach_positive_pay_rule_id"`
	AmountCondition      string          `json:"amount_condition"`
	BankAccountID        string          `json:"bank_account_id"`
	Description          string          `json:"description"`
	CreatedAt            string          `json:"created_at"`
	UpdatedAt            string          `json:"updated_at"`
	CurrencyCode         string          `json:"currency_code"`
	Status               string          `json:"status"`
	Raw                  json.RawMessage `json:"raw"`
}

type InternationalWireTransfer struct {
	AccountNumberID            string         `json:"account_number_id"`
	AllowOverdraft             bool           `json:"allow_overdraft"`
	Amount                     int64          `json:"amount"`
	BankAccountID              string         `json:"bank_account_id"`
	BeneficiaryAccountNumber   string         `json:"beneficiary_account_number"`
	BeneficiaryAddress         Address        `json:"beneficiary_address"`
	BeneficiaryFI              string         `json:"beneficiary_fi"`
	BeneficiaryName            string         `json:"beneficiary_name"`
	ChargeBearer               string         `json:"charge_bearer"`
	Charges                    []Charge       `json:"charges"`
	CompletedAt                *string        `json:"completed_at"`
	CounterpartyID             string         `json:"counterparty_id"`
	CreatedAt                  string         `json:"created_at"`
	CurrencyCode               string         `json:"currency_code"`
	Description                string         `json:"description"`
	EndToEndID                 string         `json:"end_to_end_id"`
	FXQuoteID                  string         `json:"fx_quote_id"`
	FXRate                     string         `json:"fx_rate"`
	ID                         string         `json:"id"`
	IdempotencyKey             *string        `json:"idempotency_key"`
	InitiatedAt                string         `json:"initiated_at"`
	InstructedAmount           int            `json:"instructed_amount"`
	InstructedCurrencyCode     string         `json:"instructed_currency_code"`
	InstructionID              string         `json:"instruction_id"`
	InstructionToBeneficiaryFI string         `json:"instruction_to_beneficiary_fi"`
	IsIncoming                 bool           `json:"is_incoming"`
	ManualReviewAt             *string        `json:"manual_review_at"`
	OriginatorAccountNumber    string         `json:"originator_account_number"`
	OriginatorAddress          Address        `json:"originator_address"`
	OriginatorFI               string         `json:"originator_fi"`
	OriginatorName             string         `json:"originator_name"`
	PendingSubmissionAt        string         `json:"pending_submission_at"`
	RawMessage                 *string        `json:"raw_message"`
	RemittanceInfo             RemittanceInfo `json:"remittance_info"`
	ReturnReason               *string        `json:"return_reason"`
	ReturnedAmount             *int           `json:"returned_amount"`
	ReturnedAt                 *string        `json:"returned_at"`
	ReturnedCurrencyCode       *string        `json:"returned_currency_code"`
	SettledAmount              int            `json:"settled_amount"`
	SettledCurrencyCode        string         `json:"settled_currency_code"`
	SettlementDate             string         `json:"settlement_date"`
	Status                     string         `json:"status"`
	SubmittedAt                string         `json:"submitted_at"`
	UETR                       string         `json:"uetr"`
	UltimateBeneficiaryAddress *string        `json:"ultimate_beneficiary_address"`
	UltimateBeneficiaryName    string         `json:"ultimate_beneficiary_name"`
	UltimateOriginatorAddress  *string        `json:"ultimate_originator_address"`
	UltimateOriginatorName     string         `json:"ultimate_originator_name"`
	UpdatedAt                  string         `json:"updated_at"`
}

type RemittanceInfo struct {
	GeneralInfo string `json:"general_info"`
}

func (c *client) InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	payoutType := models.ExtractNamespacedMetadata(pr.Metadata, ColumnPayoutTypeMetadataKey)

	switch payoutType {
	case "ach":
		achPayload := &ACHPayoutRequest{
			Amount:          pr.Amount,
			BankAccountID:   pr.SourceAccount,
			Description:     pr.Description,
			AchCompanyID:    pr.DestinationAccount,
			AmountCondition: models.ExtractNamespacedMetadata(pr.Metadata, ColumnAmountConditionMetadataKey),
		}

		return c.initiateACHPayout(ctx, achPayload)
	case "wire":
		wirePayload := &WirePayoutRequest{
			Amount:                           pr.Amount,
			CurrencyCode:                     pr.CurrencyCode,
			CounterPartyId:                   pr.DestinationAccount,
			Description:                      pr.Description,
			BankAccountID:                    pr.SourceAccount,
			AllowOverdraft:                   models.ExtractNamespacedMetadata(pr.Metadata, ColumnAllowOverdraftMetadataKey) == "true",
			UltimateOriginatorCounterpartyId: models.ExtractNamespacedMetadata(pr.Metadata, ColumnUltimateOriginatorCounterpartyIdMetadataKey),
			UltimateOriginatorAccountNumber:  models.ExtractNamespacedMetadata(pr.Metadata, ColumnUltimateOriginatorAccountNumberMetadataKey),
		}

		return c.initiateWirePayouts(ctx, wirePayload)
	case "international-wire":
		internationalPayload := &InternationalWirePayoutRequest{
			Amount:                   pr.Amount,
			CounterpartyID:           pr.DestinationAccount,
			CurrencyCode:             pr.CurrencyCode,
			Description:              pr.Description,
			BankAccountID:            pr.SourceAccount,
			ChargeBearer:             models.ExtractNamespacedMetadata(pr.Metadata, ColumnChargeBearerMetadataKey),
			AllowOverdraft:           models.ExtractNamespacedMetadata(pr.Metadata, ColumnAllowOverdraftMetadataKey) == "true",
			FxQuoteID:                models.ExtractNamespacedMetadata(pr.Metadata, ColumnFxQuoteIdMetadataKey),
			IntermediaryBank:         models.ExtractNamespacedMetadata(pr.Metadata, ColumnIntermediaryBankMetadataKey),
			MessageToBeneficiaryBank: models.ExtractNamespacedMetadata(pr.Metadata, ColumnMessageToBeneficiaryBankMetadataKey),
			RemittanceInformation: RemittanceInformationRequest{
				GeneralInfo:          models.ExtractNamespacedMetadata(pr.Metadata, ColumnGeneralInfoMetadataKey),
				BeneficiaryReference: models.ExtractNamespacedMetadata(pr.Metadata, ColumnBeneficiaryReferenceMetadataKey),
				PurposeCode:          models.ExtractNamespacedMetadata(pr.Metadata, ColumnPurposeCodeMetadataKey),
			},
		}

		return c.initiateInternationalPayout(ctx, internationalPayload)
	case "realtime":
		realtimePayload := &RealtimeTransferRequest{
			Amount:                       pr.Amount,
			CurrencyCode:                 pr.CurrencyCode,
			Description:                  pr.Description,
			CounterpartyID:               pr.DestinationAccount,
			BankAccountID:                pr.SourceAccount,
			AllowOverdraft:               models.ExtractNamespacedMetadata(pr.Metadata, ColumnAllowOverdraftMetadataKey) == "true",
			UltimateDebtorCounterparty:   models.ExtractNamespacedMetadata(pr.Metadata, ColumnUltimateDebtorCounterpartyMetadataKey),
			UltimateDebtorCounterpartyID: models.ExtractNamespacedMetadata(pr.Metadata, ColumnUltimateDebtorCounterpartyIdMetadataKey),
			EndToEndID:                   models.ExtractNamespacedMetadata(pr.Metadata, ColumnEndToEndIdMetadataKey),
		}

		return c.initiateRealtimePayout(ctx, realtimePayload)
	}

	return nil, nil
}

func (c *client) initiateACHPayout(ctx context.Context, pr *ACHPayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_ach_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return &PayoutResponse{}, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "ach-positive-pay-rules", bytes.NewBuffer(body))
	if err != nil {
		return &PayoutResponse{}, err
	}

	var response ACHPayoutResponse
	var errRes columnError
	if _, err := c.httpClient.Do(ctx, req, &response, errRes); err != nil {
		return &PayoutResponse{}, err
	}

	return MapAchPayout(response)
}

func (c *client) initiateWirePayouts(ctx context.Context, pr *WirePayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_wire_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return &PayoutResponse{}, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "transfers/wire", bytes.NewBuffer(body))
	if err != nil {
		return &PayoutResponse{}, err
	}

	var response WirePayoutResponse

	if _, err := c.httpClient.Do(ctx, req, &response, nil); err != nil {
		return &PayoutResponse{}, err
	}

	return MapWirePayout(response)
}

func (c *client) initiateInternationalPayout(ctx context.Context, pr *InternationalWirePayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_international_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return &PayoutResponse{}, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "transfers/international", bytes.NewBuffer(body))
	if err != nil {
		return &PayoutResponse{}, err
	}

	var response InternationalWirePayoutResponse

	if _, err := c.httpClient.Do(ctx, req, &response, nil); err != nil {
		return &PayoutResponse{}, err
	}

	return MapInternationalWirePayout(response)
}

func (c *client) initiateRealtimePayout(ctx context.Context, pr *RealtimeTransferRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_realtime_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return &PayoutResponse{}, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "transfers/realtime", bytes.NewBuffer(body))
	if err != nil {
		return &PayoutResponse{}, err
	}

	var response RealtimeTransferResponse

	if _, err := c.httpClient.Do(ctx, req, &response, nil); err != nil {
		return &PayoutResponse{}, err
	}

	return MapRealtimePayout(response)
}

func MapInternationalWirePayout(response InternationalWirePayoutResponse) (*PayoutResponse, error) {
	raw, err := json.Marshal(response)
	if err != nil {
		return &PayoutResponse{}, err
	}

	return &PayoutResponse{
		ID:            response.ID,
		Amount:        response.Amount,
		BankAccountID: response.BankAccountID,
		Description:   response.Description,
		CurrencyCode:  response.CurrencyCode,
		CreatedAt:     response.CreatedAt,
		UpdatedAt:     response.UpdatedAt,
		Raw:           raw,
	}, nil
}

func MapRealtimePayout(response RealtimeTransferResponse) (*PayoutResponse, error) {
	raw, err := json.Marshal(response)
	if err != nil {
		return &PayoutResponse{}, err
	}

	return &PayoutResponse{
		ID:            response.ID,
		Amount:        response.Amount,
		BankAccountID: response.BankAccountID,
		Description:   response.Description,
		CurrencyCode:  response.CurrencyCode,
		Raw:           raw,
	}, nil
}

func MapWirePayout(response WirePayoutResponse) (*PayoutResponse, error) {
	raw, err := json.Marshal(response)
	if err != nil {
		return &PayoutResponse{}, err
	}

	return &PayoutResponse{
		ID:            response.ID,
		Amount:        response.Amount,
		BankAccountID: response.BankAccountID,
		Description:   response.Description,
		CurrencyCode:  response.CurrencyCode,
		CreatedAt:     response.CreatedAt,
		UpdatedAt:     response.UpdatedAt,
		Raw:           raw,
	}, nil
}

func MapAchPayout(response ACHPayoutResponse) (*PayoutResponse, error) {
	raw, err := json.Marshal(response)
	if err != nil {
		return &PayoutResponse{}, err
	}

	return &PayoutResponse{
		ID:                   response.ACHPositivePayRuleID,
		Amount:               response.Amount,
		AmountCondition:      response.AmountCondition,
		BankAccountID:        response.BankAccountID,
		Description:          response.Description,
		AchPositivePayRuleID: response.ACHPositivePayRuleID,
		Raw:                  raw,
	}, nil
}
