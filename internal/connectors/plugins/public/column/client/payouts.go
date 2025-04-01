package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
)

type ACHPayoutRequest struct {
	Description                       string `json:"description,omitempty"`
	Amount                            int64  `json:"amount,omitempty"`
	BankAccountID                     string `json:"bank_account_id"`
	CounterPartyId                    string `json:"counterparty_id"`
	Type                              string `json:"type"`
	CurrencyCode                      string `json:"currency_code"`
	EffectiveDate                     string `json:"effective_date,omitempty"`
	SameDay                           string `json:"same_day,omitempty"`
	CompanyDiscretionaryData          string `json:"company_discretionary_data,omitempty"`
	CompanyEntryDescription           string `json:"company_entry_description,omitempty"`
	CompanyName                       string `json:"company_name,omitempty"`
	PaymentRelatedInfo                string `json:"payment_related_info,omitempty"`
	ReceiverName                      string `json:"receiver_name,omitempty"`
	ReceiverId                        string `json:"receiver_id,omitempty"`
	EntryClassCode                    string `json:"entry_class_code"`
	AllowOverdraft                    bool   `json:"allow_overdraft,omitempty"`
	UltimateBeneficiaryCounterparty   string `json:"ultimate_beneficiary_counterparty,omitempty"`
	UltimateBeneficiaryCounterpartyId string `json:"ultimate_beneficiary_counterparty_id,omitempty"`
	UltimateOriginatorAccountNumber   string `json:"ultimate_originator_counterparty,omitempty"`
	UltimateOriginatorCounterpartyId  string `json:"ultimate_originator_counterparty_id,omitempty"`
}

type ACHPayoutResponse struct {
	ID                                string  `json:"id"`
	IAT                               *string `json:"iat"`
	Type                              string  `json:"type"`
	Amount                            int64   `json:"amount"`
	Status                            string  `json:"status"`
	IsOnUs                            bool    `json:"is_on_us"`
	SameDay                           bool    `json:"same_day"`
	CompanyID                         string  `json:"company_id"`
	CreatedAt                         string  `json:"created_at"`
	SettledAt                         *string `json:"settled_at"`
	UpdatedAt                         string  `json:"updated_at"`
	Description                       string  `json:"description"`
	IsIncoming                        bool    `json:"is_incoming"`
	ReceiverID                        string  `json:"receiver_id"`
	ReturnedAt                        *string `json:"returned_at"`
	CancelledAt                       *string `json:"cancelled_at"`
	CompanyName                       string  `json:"company_name"`
	CompletedAt                       *string `json:"completed_at"`
	EffectiveOn                       string  `json:"effective_on"`
	InitiatedAt                       string  `json:"initiated_at"`
	NSFDeadline                       *string `json:"nsf_deadline"`
	SubmittedAt                       string  `json:"submitted_at"`
	TraceNumber                       string  `json:"trace_number"`
	CurrencyCode                      string  `json:"currency_code"`
	ReceiverName                      string  `json:"receiver_name"`
	ReturnDetails                     []any   `json:"return_details"`
	AcknowledgedAt                    *string `json:"acknowledged_at"`
	AllowOverdraft                    bool    `json:"allow_overdraft"`
	BankAccountID                     string  `json:"bank_account_id"`
	CounterpartyID                    string  `json:"counterparty_id"`
	IdempotencyKey                    string  `json:"idempotency_key"`
	EntryClassCode                    string  `json:"entry_class_code"`
	ManualReviewAt                    *string `json:"manual_review_at"`
	AccountNumberID                   string  `json:"account_number_id"`
	FundsAvailability                 string  `json:"funds_availability"`
	ODFIRoutingNumber                 string  `json:"odfi_routing_number"`
	ReturnContestedAt                 *string `json:"return_contested_at"`
	PaymentRelatedInfo                string  `json:"payment_related_info"`
	ReturnDishonoredAt                *string `json:"return_dishonored_at"`
	ReturnDishonoredFundsUnlockedAt   *string `json:"return_dishonored_funds_unlocked_at"`
	NotificationOfChanges             *any    `json:"notification_of_changes"`
	CompanyEntryDescription           *string `json:"company_entry_description"`
	ReversalPairTransferID            *string `json:"reversal_pair_transfer_id"`
	CompanyDiscretionaryData          *string `json:"company_discretionary_data"`
	UltimateBeneficiaryCounterpartyID *string `json:"ultimate_beneficiary_counterparty_id"`
	UltimateOriginatorCounterpartyID  *string `json:"ultimate_originator_counterparty_id"`
}
type WirePayoutRequest struct {
	Amount                           int64  `json:"amount"`
	CurrencyCode                     string `json:"currency_code"`
	BankAccountID                    string `json:"bank_account_id"`
	CounterPartyId                   string `json:"counterparty_id"`
	AllowOverdraft                   bool   `json:"allow_overdraft,omitempty"`
	Description                      string `json:"description,omitempty"`
	UltimateOriginatorCounterpartyId string `json:"ultimate_originator_counterparty_id,omitempty"`
	UltimateOriginatorAccountNumber  string `json:"ultimate_originator_account_number,omitempty"`
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
	AllowOverdraft           bool                  `json:"allow_overdraft,omitempty"`
	Amount                   int64                 `json:"amount"`
	BankAccountID            string                `json:"bank_account_id"`
	ChargeBearer             string                `json:"charge_bearer,omitempty"`
	CounterpartyID           string                `json:"counterparty_id"`
	CurrencyCode             string                `json:"currency_code"`
	Description              string                `json:"description,omitempty"`
	FxQuoteID                string                `json:"fx_quote_id,omitempty"`
	IntermediaryBank         string                `json:"intermediary_bank,omitempty"`
	MessageToBeneficiaryBank string                `json:"message_to_beneficiary_bank,omitempty"`
	RemittanceInfo           RemittanceInfoRequest `json:"remittance_info,omitempty"`
}

type RemittanceInfoRequest struct {
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
	ID                         string                `json:"id"`
	CreatedAt                  string                `json:"created_at"`
	UpdatedAt                  string                `json:"updated_at"`
	InitiatedAt                string                `json:"initiated_at"`
	PendingSubmissionAt        string                `json:"pending_submission_at"`
	SubmittedAt                string                `json:"submitted_at"`
	CompletedAt                *string               `json:"completed_at"`
	ManualReviewAt             *string               `json:"manual_review_at"`
	ReturnedAt                 *string               `json:"returned_at"`
	IdempotencyKey             *string               `json:"idempotency_key"`
	RawMessage                 *string               `json:"raw_message"`
	ReturnReason               *string               `json:"return_reason"`
	ReturnedAmount             *int64                `json:"returned_amount"`
	ReturnedCurrencyCode       *string               `json:"returned_currency_code"`
	SettlementDate             string                `json:"settlement_date"`
	UETR                       string                `json:"uetr"`
	InstructionID              string                `json:"instruction_id"`
	EndToEndID                 string                `json:"end_to_end_id"`
	Status                     string                `json:"status"`
	AccountNumberID            string                `json:"account_number_id"`
	BankAccountID              string                `json:"bank_account_id"`
	CounterpartyID             string                `json:"counterparty_id"`
	FXQuoteID                  string                `json:"fx_quote_id"`
	FXRate                     string                `json:"fx_rate"`
	Amount                     int64                 `json:"amount"`
	CurrencyCode               string                `json:"currency_code"`
	InstructedAmount           int64                 `json:"instructed_amount"`
	InstructedCurrencyCode     string                `json:"instructed_currency_code"`
	SettledAmount              int64                 `json:"settled_amount"`
	SettledCurrencyCode        string                `json:"settled_currency_code"`
	AllowOverdraft             bool                  `json:"allow_overdraft"`
	IsIncoming                 bool                  `json:"is_incoming"`
	Description                string                `json:"description"`
	InstructionToBeneficiaryFI string                `json:"instruction_to_beneficiary_fi"`
	BeneficiaryAccountNumber   string                `json:"beneficiary_account_number"`
	BeneficiaryFI              string                `json:"beneficiary_fi"`
	BeneficiaryName            string                `json:"beneficiary_name"`
	OriginatorAccountNumber    string                `json:"originator_account_number"`
	OriginatorFI               string                `json:"originator_fi"`
	OriginatorName             string                `json:"originator_name"`
	UltimateBeneficiaryName    string                `json:"ultimate_beneficiary_name"`
	UltimateOriginatorName     string                `json:"ultimate_originator_name"`
	BeneficiaryAddress         Address               `json:"beneficiary_address"`
	OriginatorAddress          Address               `json:"originator_address"`
	UltimateBeneficiaryAddress *Address              `json:"ultimate_beneficiary_address"`
	UltimateOriginatorAddress  *Address              `json:"ultimate_originator_address"`
	ChargeBearer               string                `json:"charge_bearer"`
	Charges                    []Charge              `json:"charges"`
	RemittanceInfo             RemittanceInfoRequest `json:"remittance_info"`
}

type RealtimeTransferRequest struct {
	Amount                       int64  `json:"amount"`
	CurrencyCode                 string `json:"currency_code"`
	BankAccountID                string `json:"bank_account_id"`
	Description                  string `json:"description,omitempty"`
	CounterpartyID               string `json:"counterparty_id"`
	AllowOverdraft               bool   `json:"allow_overdraft,omitempty"`
	UltimateDebtorCounterparty   string `json:"ultimate_debtor_counterparty,omitempty"`
	UltimateDebtorCounterpartyID string `json:"ultimate_debtor_counterparty_id,omitempty"`
	EndToEndID                   string `json:"end_to_end_id,omitempty"`
}

type RealtimeTransferResponse struct {
	AcceptedAt                   *string `json:"accepted_at"`
	AccountNumberID              string  `json:"account_number_id,omitempty"`
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
	ID             string            `json:"id"`
	Amount         int64             `json:"amount"`
	BankAccountID  string            `json:"bank_account_id"`
	CounterpartyId string            `json:"counterparty_id"`
	Description    string            `json:"description"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	CurrencyCode   string            `json:"currency_code"`
	Status         string            `json:"status"`
	IsIncoming     bool              `json:"is_incoming"`
	Metadata       map[string]string `json:"metadata"`
	Raw            json.RawMessage   `json:"raw"`
}

type InternationalWireTransfer struct {
	AccountNumberID            string                `json:"account_number_id"`
	AllowOverdraft             bool                  `json:"allow_overdraft"`
	Amount                     int64                 `json:"amount"`
	BankAccountID              string                `json:"bank_account_id"`
	BeneficiaryAccountNumber   string                `json:"beneficiary_account_number"`
	BeneficiaryAddress         Address               `json:"beneficiary_address"`
	BeneficiaryFI              string                `json:"beneficiary_fi"`
	BeneficiaryName            string                `json:"beneficiary_name"`
	ChargeBearer               string                `json:"charge_bearer"`
	Charges                    []Charge              `json:"charges"`
	CompletedAt                *string               `json:"completed_at"`
	CounterpartyID             string                `json:"counterparty_id"`
	CreatedAt                  string                `json:"created_at"`
	CurrencyCode               string                `json:"currency_code"`
	Description                string                `json:"description"`
	EndToEndID                 string                `json:"end_to_end_id"`
	FXQuoteID                  string                `json:"fx_quote_id"`
	FXRate                     string                `json:"fx_rate"`
	ID                         string                `json:"id"`
	IdempotencyKey             *string               `json:"idempotency_key"`
	InitiatedAt                string                `json:"initiated_at"`
	InstructedAmount           int                   `json:"instructed_amount"`
	InstructedCurrencyCode     string                `json:"instructed_currency_code"`
	InstructionID              string                `json:"instruction_id"`
	InstructionToBeneficiaryFI string                `json:"instruction_to_beneficiary_fi"`
	IsIncoming                 bool                  `json:"is_incoming"`
	ManualReviewAt             *string               `json:"manual_review_at"`
	OriginatorAccountNumber    string                `json:"originator_account_number"`
	OriginatorAddress          Address               `json:"originator_address"`
	OriginatorFI               string                `json:"originator_fi"`
	OriginatorName             string                `json:"originator_name"`
	PendingSubmissionAt        string                `json:"pending_submission_at"`
	RawMessage                 *string               `json:"raw_message"`
	RemittanceInfo             RemittanceInfoRequest `json:"remittance_info"`
	ReturnReason               *string               `json:"return_reason"`
	ReturnedAmount             *int                  `json:"returned_amount"`
	ReturnedAt                 *string               `json:"returned_at"`
	ReturnedCurrencyCode       *string               `json:"returned_currency_code"`
	SettledAmount              int                   `json:"settled_amount"`
	SettledCurrencyCode        string                `json:"settled_currency_code"`
	SettlementDate             string                `json:"settlement_date"`
	Status                     string                `json:"status"`
	SubmittedAt                string                `json:"submitted_at"`
	UETR                       string                `json:"uetr"`
	UltimateBeneficiaryAddress *string               `json:"ultimate_beneficiary_address"`
	UltimateBeneficiaryName    string                `json:"ultimate_beneficiary_name"`
	UltimateOriginatorAddress  *string               `json:"ultimate_originator_address"`
	UltimateOriginatorName     string                `json:"ultimate_originator_name"`
	UpdatedAt                  string                `json:"updated_at"`
}

func (c *client) InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	payoutType := models.ExtractNamespacedMetadata(pr.Metadata, ColumnPayoutTypeMetadataKey)

	switch payoutType {
	case "ach":
		achPayload := &ACHPayoutRequest{
			Amount:                            pr.Amount,
			BankAccountID:                     pr.SourceAccount,
			Description:                       pr.Description,
			CounterPartyId:                    pr.DestinationAccount,
			CurrencyCode:                      pr.CurrencyCode,
			Type:                              models.ExtractNamespacedMetadata(pr.Metadata, ColumnTypeMetadataKey),
			EffectiveDate:                     models.ExtractNamespacedMetadata(pr.Metadata, ColumnEffectiveDateMetadataKey),
			SameDay:                           models.ExtractNamespacedMetadata(pr.Metadata, ColumnSameDayMetadataKey),
			CompanyDiscretionaryData:          models.ExtractNamespacedMetadata(pr.Metadata, ColumnCompanyDiscretionaryDataMetadataKey),
			CompanyEntryDescription:           models.ExtractNamespacedMetadata(pr.Metadata, ColumnCompanyEntryDescriptionMetadataKey),
			CompanyName:                       models.ExtractNamespacedMetadata(pr.Metadata, ColumnCompanyNameMetadataKey),
			PaymentRelatedInfo:                models.ExtractNamespacedMetadata(pr.Metadata, ColumnPaymentRelatedInfoMetadataKey),
			ReceiverName:                      models.ExtractNamespacedMetadata(pr.Metadata, ColumnReceiverNameMetadataKey),
			ReceiverId:                        models.ExtractNamespacedMetadata(pr.Metadata, ColumnReceiverIDMetadataKey),
			EntryClassCode:                    models.ExtractNamespacedMetadata(pr.Metadata, ColumnEntryClassCodeMetadataKey),
			AllowOverdraft:                    models.ExtractNamespacedMetadata(pr.Metadata, ColumnAllowOverdraftMetadataKey) == "true",
			UltimateBeneficiaryCounterparty:   models.ExtractNamespacedMetadata(pr.Metadata, ColumnUltimateBeneficiaryCounterpartyMetadataKey),
			UltimateBeneficiaryCounterpartyId: models.ExtractNamespacedMetadata(pr.Metadata, ColumnUltimateBeneficiaryCounterpartyIDMetadataKey),
			UltimateOriginatorAccountNumber:   models.ExtractNamespacedMetadata(pr.Metadata, ColumnUltimateOriginatorAccountNumberMetadataKey),
			UltimateOriginatorCounterpartyId:  models.ExtractNamespacedMetadata(pr.Metadata, ColumnUltimateOriginatorCounterpartyIdMetadataKey),
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
			RemittanceInfo: RemittanceInfoRequest{
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
		return &PayoutResponse{}, fmt.Errorf("failed to marshal ach transfer request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "transfers/ach", bytes.NewBuffer(body))
	if err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to create ach transfer request: %w", err)
	}

	var response ACHPayoutResponse
	var errRes columnError
	if _, err := c.httpClient.Do(ctx, req, &response, &errRes); err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to create ach transfer payout: %w %w", err, errRes.Error())
	}

	return MapAchPayout(response)
}

func (c *client) initiateWirePayouts(ctx context.Context, pr *WirePayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_wire_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to marshal wire transfer request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "transfers/wire", bytes.NewBuffer(body))
	if err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to create wire transfer request: %w", err)
	}

	var response WirePayoutResponse
	var errRes columnError
	if _, err := c.httpClient.Do(ctx, req, &response, &errRes); err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to create wire transfer payout: %w %w", err, errRes.Error())
	}

	return MapWirePayout(response)
}

func (c *client) initiateInternationalPayout(ctx context.Context, pr *InternationalWirePayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_international_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to marshal international transfer request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "transfers/international-wire", bytes.NewBuffer(body))
	if err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to create international transfer request: %w", err)
	}

	var response InternationalWirePayoutResponse
	var errRes columnError
	if _, err := c.httpClient.Do(ctx, req, &response, &errRes); err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to create international transfer payout: %w %w", err, errRes.Error())
	}

	return MapInternationalWirePayout(response)
}

func (c *client) initiateRealtimePayout(ctx context.Context, pr *RealtimeTransferRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_realtime_payout")

	body, err := json.Marshal(pr)
	if err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to marshal rtp transfer request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "transfers/realtime", bytes.NewBuffer(body))
	if err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to create rtp transfer request: %w", err)
	}

	var response RealtimeTransferResponse
	var errRes columnError
	if _, err := c.httpClient.Do(ctx, req, &response, &errRes); err != nil {
		return &PayoutResponse{}, fmt.Errorf("failed to create rtp transfer payout: %w %w", err, errRes.Error())
	}

	return MapRealtimePayout(response)
}

func MapInternationalWirePayout(response InternationalWirePayoutResponse) (*PayoutResponse, error) {
	raw, err := json.Marshal(response)
	if err != nil {
		return &PayoutResponse{}, err
	}

	metadata := map[string]string{
		ColumnInitiatedAtMetadataKey:                response.InitiatedAt,
		ColumnPendingSubmissionAtMetadataKey:        response.PendingSubmissionAt,
		ColumnSubmittedAtMetadataKey:                response.SubmittedAt,
		ColumnCompletedAtMetadataKey:                stringPtrToString(response.CompletedAt),
		ColumnManualReviewAtMetadataKey:             stringPtrToString(response.ManualReviewAt),
		ColumnReturnedAtMetadataKey:                 stringPtrToString(response.ReturnedAt),
		ColumnIdempotencyKeyMetadataKey:             stringPtrToString(response.IdempotencyKey),
		ColumnRawMessageMetadataKey:                 stringPtrToString(response.RawMessage),
		ColumnReturnReasonMetadataKey:               stringPtrToString(response.ReturnReason),
		ColumnReturnedAmountMetadataKey:             intPtrToString(response.ReturnedAmount),
		ColumnReturnedCurrencyCodeMetadataKey:       stringPtrToString(response.ReturnedCurrencyCode),
		ColumnSettlementDateMetadataKey:             response.SettlementDate,
		ColumnUETRMetadataKey:                       response.UETR,
		ColumnInstructionIDMetadataKey:              response.InstructionID,
		ColumnEndToEndIDMetadataKey:                 response.EndToEndID,
		ColumnAccountNumberIDMetadataKey:            response.AccountNumberID,
		ColumnFXQuoteIDMetadataKey:                  response.FXQuoteID,
		ColumnFXRateMetadataKey:                     response.FXRate,
		ColumnInstructedAmountMetadataKey:           fmt.Sprintf("%d", response.InstructedAmount),
		ColumnInstructedCurrencyCodeMetadataKey:     response.InstructedCurrencyCode,
		ColumnSettledAmountMetadataKey:              fmt.Sprintf("%d", response.SettledAmount),
		ColumnSettledCurrencyCodeMetadataKey:        response.SettledCurrencyCode,
		ColumnAllowOverdraftMetadataKey:             fmt.Sprintf("%t", response.AllowOverdraft),
		ColumnIsIncomingMetadataKey:                 fmt.Sprintf("%t", response.IsIncoming),
		ColumnInstructionToBeneficiaryFIMetadataKey: response.InstructionToBeneficiaryFI,
		ColumnBeneficiaryAccountNumberMetadataKey:   response.BeneficiaryAccountNumber,
		ColumnBeneficiaryFIMetadataKey:              response.BeneficiaryFI,
		ColumnBeneficiaryNameMetadataKey:            response.BeneficiaryName,
		ColumnOriginatorAccountNumberMetadataKey:    response.OriginatorAccountNumber,
		ColumnOriginatorFIMetadataKey:               response.OriginatorFI,
		ColumnOriginatorNameMetadataKey:             response.OriginatorName,
		ColumnUltimateBeneficiaryNameMetadataKey:    response.UltimateBeneficiaryName,
		ColumnUltimateOriginatorNameMetadataKey:     response.UltimateOriginatorName,
		ColumnChargeBearerMetadataKey:               response.ChargeBearer,
	}

	if len(response.Charges) > 0 {
		chargesJson, err := json.Marshal(response.Charges)
		if err == nil {
			metadata[ColumnChargesMetadataKey] = string(chargesJson)
		}
	}

	if response.RemittanceInfo.GeneralInfo != "" || response.RemittanceInfo.BeneficiaryReference != "" || response.RemittanceInfo.PurposeCode != "" {
		remittanceJson, err := json.Marshal(response.RemittanceInfo)
		if err == nil {
			metadata[ColumnRemittanceInfoMetadataKey] = string(remittanceJson)
		}
	}

	// Handle address objects
	beneficiaryAddressJson, err := json.Marshal(response.BeneficiaryAddress)
	if err == nil {
		metadata[ColumnBeneficiaryAddressMetadataKey] = string(beneficiaryAddressJson)
	}

	originatorAddressJson, err := json.Marshal(response.OriginatorAddress)
	if err == nil {
		metadata[ColumnOriginatorAddressMetadataKey] = string(originatorAddressJson)
	}

	if response.UltimateBeneficiaryAddress != nil {
		ultimateBeneficiaryAddressJson, err := json.Marshal(*response.UltimateBeneficiaryAddress)
		if err == nil {
			metadata[ColumnUltimateBeneficiaryAddressMetadataKey] = string(ultimateBeneficiaryAddressJson)
		}
	}

	if response.UltimateOriginatorAddress != nil {
		ultimateOriginatorAddressJson, err := json.Marshal(*response.UltimateOriginatorAddress)
		if err == nil {
			metadata[ColumnUltimateOriginatorAddressMetadataKey] = string(ultimateOriginatorAddressJson)
		}
	}

	return &PayoutResponse{
		ID:             response.ID,
		Amount:         response.Amount,
		BankAccountID:  response.BankAccountID,
		CounterpartyId: response.CounterpartyID,
		Description:    response.Description,
		CurrencyCode:   response.CurrencyCode,
		CreatedAt:      response.CreatedAt,
		UpdatedAt:      response.UpdatedAt,
		Status:         response.Status,
		IsIncoming:     response.IsIncoming,
		Metadata:       metadata,
		Raw:            raw,
	}, nil
}

func MapRealtimePayout(response RealtimeTransferResponse) (*PayoutResponse, error) {
	raw, err := json.Marshal(response)
	if err != nil {
		return &PayoutResponse{}, err
	}

	metadata := map[string]string{
		ColumnAcceptedAtMetadataKey:                   stringPtrToString(response.AcceptedAt),
		ColumnAccountNumberIDMetadataKey:              response.AccountNumberID,
		ColumnAllowOverdraftMetadataKey:               fmt.Sprintf("%t", response.AllowOverdraft),
		ColumnBlockedAtMetadataKey:                    stringPtrToString(response.BlockedAt),
		ColumnCompletedAtMetadataKey:                  stringPtrToString(response.CompletedAt),
		ColumnUltimateDebtorCounterpartyIdMetadataKey: response.UltimateDebtorCounterpartyID,
		ColumnIdempotencyKeyMetadataKey:               stringPtrToString(response.IdempotencyKey),
		ColumnIsIncomingMetadataKey:                   fmt.Sprintf("%t", response.IsIncoming),
		ColumnIsOnUsMetadataKey:                       fmt.Sprintf("%t", response.IsOnUs),
		ColumnManualReviewApprovedAtMetadataKey:       stringPtrToString(response.ManualReviewApprovedAt),
		ColumnManualReviewAtMetadataKey:               stringPtrToString(response.ManualReviewAt),
		ColumnManualReviewRejectedAtMetadataKey:       stringPtrToString(response.ManualReviewRejectedAt),
		ColumnPendingAtMetadataKey:                    stringPtrToString(response.PendingAt),
		ColumnRejectedAtMetadataKey:                   stringPtrToString(response.RejectedAt),
		ColumnRejectedCodeMetadataKey:                 stringPtrToString(response.RejectedCode),
		ColumnRejectionCodeDescriptionMetadataKey:     stringPtrToString(response.RejectionCodeDescription),
		ColumnRejectionAdditionalInfoMetadataKey:      stringPtrToString(response.RejectionAdditionalInfo),
		ColumnReturnPairTransferIDMetadataKey:         stringPtrToString(response.ReturnPairTransferID),
	}

	return &PayoutResponse{
		ID:             response.ID,
		Amount:         response.Amount,
		BankAccountID:  response.BankAccountID,
		CounterpartyId: response.CounterpartyID,
		IsIncoming:     response.IsIncoming,
		Description:    response.Description,
		CurrencyCode:   response.CurrencyCode,
		CreatedAt:      response.InitiatedAt,
		Status:         response.Status,
		Metadata:       metadata,
		Raw:            raw,
	}, nil
}

func MapWirePayout(response WirePayoutResponse) (*PayoutResponse, error) {
	raw, err := json.Marshal(response)
	if err != nil {
		return &PayoutResponse{}, err
	}

	metadata := map[string]string{
		ColumnInitiatedAtMetadataKey:           response.InitiatedAt,
		ColumnPendingSubmissionAtMetadataKey:   response.PendingSubmissionAt,
		ColumnSubmittedAtMetadataKey:           response.SubmittedAt,
		ColumnCompletedAtMetadataKey:           response.CompletedAt,
		ColumnRejectedAtMetadataKey:            response.RejectedAt,
		ColumnIdempotencyKeyMetadataKey:        response.IdempotencyKey,
		ColumnAllowOverdraftMetadataKey:        fmt.Sprintf("%t", response.AllowOverdraft),
		ColumnPlatformIDMetadataKey:            response.PlatformID,
		ColumnIsOnUsMetadataKey:                fmt.Sprintf("%t", response.IsOnUs),
		ColumnIsIncomingMetadataKey:            fmt.Sprintf("%t", response.IsIncoming),
		ColumnWireDrawdownRequestIDMetadataKey: response.WireDrawdownRequestID,
	}

	return &PayoutResponse{
		ID:             response.ID,
		Amount:         response.Amount,
		BankAccountID:  response.BankAccountID,
		CounterpartyId: response.CounterpartyID,
		Description:    response.Description,
		CurrencyCode:   response.CurrencyCode,
		CreatedAt:      response.CreatedAt,
		UpdatedAt:      response.UpdatedAt,
		Status:         response.Status,
		IsIncoming:     response.IsIncoming,
		Metadata:       metadata,
		Raw:            raw,
	}, nil
}

func MapAchPayout(response ACHPayoutResponse) (*PayoutResponse, error) {
	raw, err := json.Marshal(response)

	if err != nil {
		return &PayoutResponse{}, err
	}

	metadata := map[string]string{
		ColumnTypeMetadataKey:                              response.Type,
		ColumnIsOnUsMetadataKey:                            fmt.Sprintf("%t", response.IsOnUs),
		ColumnSameDayMetadataKey:                           fmt.Sprintf("%t", response.SameDay),
		ColumnCompanyIDMetadataKey:                         response.CompanyID,
		ColumnSettledAtMetadataKey:                         stringPtrToString(response.SettledAt),
		ColumnIsIncomingMetadataKey:                        fmt.Sprintf("%t", response.IsIncoming),
		ColumnReceiverIDMetadataKey:                        response.ReceiverID,
		ColumnReturnedAtMetadataKey:                        stringPtrToString(response.ReturnedAt),
		ColumnCancelledAtMetadataKey:                       stringPtrToString(response.CancelledAt),
		ColumnCompanyNameMetadataKey:                       response.CompanyName,
		ColumnCompletedAtMetadataKey:                       stringPtrToString(response.CompletedAt),
		ColumnEffectiveDateMetadataKey:                     response.EffectiveOn,
		ColumnInitiatedAtMetadataKey:                       response.InitiatedAt,
		ColumnNSFDeadlineMetadataKey:                       stringPtrToString(response.NSFDeadline),
		ColumnSubmittedAtMetadataKey:                       response.SubmittedAt,
		ColumnTraceNumberMetadataKey:                       response.TraceNumber,
		ColumnReceiverNameMetadataKey:                      response.ReceiverName,
		ColumnAcknowledgedAtMetadataKey:                    stringPtrToString(response.AcknowledgedAt),
		ColumnAllowOverdraftMetadataKey:                    fmt.Sprintf("%t", response.AllowOverdraft),
		ColumnIdempotencyKeyMetadataKey:                    response.IdempotencyKey,
		ColumnEntryClassCodeMetadataKey:                    response.EntryClassCode,
		ColumnManualReviewAtMetadataKey:                    stringPtrToString(response.ManualReviewAt),
		ColumnAccountNumberIDMetadataKey:                   response.AccountNumberID,
		ColumnFundsAvailabilityMetadataKey:                 response.FundsAvailability,
		ColumnODFIRoutingNumberMetadataKey:                 response.ODFIRoutingNumber,
		ColumnReturnContestedAtMetadataKey:                 stringPtrToString(response.ReturnContestedAt),
		ColumnPaymentRelatedInfoMetadataKey:                response.PaymentRelatedInfo,
		ColumnReturnDishonoredAtMetadataKey:                stringPtrToString(response.ReturnDishonoredAt),
		ColumnReturnDishonoredFundsUnlockedAtMetadataKey:   stringPtrToString(response.ReturnDishonoredFundsUnlockedAt),
		ColumnCompanyEntryDescriptionMetadataKey:           stringPtrToString(response.CompanyEntryDescription),
		ColumnReversalPairTransferIDMetadataKey:            stringPtrToString(response.ReversalPairTransferID),
		ColumnCompanyDiscretionaryDataMetadataKey:          stringPtrToString(response.CompanyDiscretionaryData),
		ColumnUltimateBeneficiaryCounterpartyIDMetadataKey: stringPtrToString(response.UltimateBeneficiaryCounterpartyID),
		ColumnUltimateOriginatorCounterpartyIDMetadataKey:  stringPtrToString(response.UltimateOriginatorCounterpartyID),
	}

	if response.IAT != nil {
		metadata[ColumnIATMetadataKey] = *response.IAT
	}

	return &PayoutResponse{
		ID:             response.ID,
		Amount:         response.Amount,
		BankAccountID:  response.BankAccountID,
		CounterpartyId: response.CounterpartyID,
		Description:    response.Description,
		CurrencyCode:   response.CurrencyCode,
		CreatedAt:      response.CreatedAt,
		UpdatedAt:      response.UpdatedAt,
		Status:         response.Status,
		IsIncoming:     response.IsIncoming,
		Metadata:       metadata,
		Raw:            raw,
	}, nil
}

func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtrToString(i *int64) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%d", *i)
}
