package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"net/http"
	"time"
)

type CounterpartyDetails struct {
	CounterpartyAccountNumber        string `json:"counterparty_account_number,omitempty"`
	CounterpartyAccountNumberFormat  string `json:"counterparty_account_number_format,omitempty"`
	CounterpartyBankIdentifier       string `json:"counterparty_bank_identifier,omitempty"`
	CounterpartyBankIdentifierFormat string `json:"counterparty_bank_identifier_format,omitempty"`
}

type CheckDetails struct {
	CheckNumber string `json:"check_number,omitempty"`
	CheckKey    string `json:"check_key,omitempty"`
}

type PagodaPaymentDetails struct {
	NoticeNumber       string `json:"notice_number,omitempty"`
	CreditorFiscalCode string `json:"creditor_fiscal_code,omitempty"`
	Iuv                string `json:"iuv,omitempty"`
}

type DirectDebitHoldDetails struct {
	GuardingRate string `json:"guarding_rate,omitempty"`
}

type FinancingInstallmentDetails struct {
	TotalInstallmentNumber   int64 `json:"total_installment_number,omitempty"`
	CurrentInstallmentNumber int64 `json:"current_installment_number,omitempty"`
}

type LogoDetails struct {
	Small  string `json:"small,omitempty"`
	Medium string `json:"medium,omitempty"`
}

type Transactions struct {
	Id                    string                       `json:"id"`
	TransactionId         string                       `json:"transaction_id"`
	Amount                json.Number                  `json:"amount"`
	AmountCents           int64                        `json:"amount_cents"`
	SettledBalance        json.Number                  `json:"settled_balance"`
	SettledBalanceCents   int64                        `json:"settled_balance_cents"`
	AttachmentsIds        *[]string                    `json:"attachments_ids,omitempty"`
	Logo                  *LogoDetails                 `json:"logo,omitempty"`
	LocalAmount           json.Number                  `json:"local_amount,omitempty"`
	LocalAmountCents      int64                        `json:"local_amount_cents,omitempty"`
	Side                  string                       `json:"side"`
	OperationType         string                       `json:"operation_type"`
	Currency              string                       `json:"currency"`
	LocalCurrency         string                       `json:"local_currency"`
	Label                 string                       `json:"label"`
	CleanCounterpartyName string                       `json:"clean_counterparty_name"`
	SettledAt             string                       `json:"settled_at,omitempty"`
	EmittedAt             string                       `json:"emitted_at"`
	UpdatedAt             string                       `json:"updated_at"`
	Status                string                       `json:"status"`
	Note                  string                       `json:"note,omitempty"`
	Reference             string                       `json:"reference,omitempty"`
	VatAmount             json.Number                  `json:"vat_amount,omitempty"`
	VatAmountCents        int64                        `json:"vat_amount_cents,omitempty"`
	VatRate               json.Number                  `json:"vat_rate,omitempty"`
	InitiatorId           string                       `json:"initiator_id"`
	LabelIds              *[]string                    `json:"label_ids,omitempty"`
	AttachmentLost        bool                         `json:"attachment_lost"`
	AttachmentRequired    bool                         `json:"attachment_required"`
	CardLastDigits        string                       `json:"card_last_digits,omitempty"`
	Category              string                       `json:"category"`
	SubjectType           string                       `json:"subject_type"`
	BankAccountId         string                       `json:"bank_account_id"`
	IsExternalTransaction bool                         `json:"is_external_transaction"`
	Transfer              *CounterpartyDetails         `json:"transfer,omitempty"`
	Income                *CounterpartyDetails         `json:"income,omitempty"`
	SwiftIncome           *CounterpartyDetails         `json:"swift_income,omitempty"`
	DirectDebit           *CounterpartyDetails         `json:"direct_debit,omitempty"`
	Check                 *CheckDetails                `json:"check,omitempty"`
	FinancingInstallment  *FinancingInstallmentDetails `json:"financing_installment,omitempty"`
	PagodaPayment         *PagodaPaymentDetails        `json:"pagoda_payment,omitempty"`
	DirectDebitCollection *CounterpartyDetails         `json:"direct_debit_collection,omitempty"`
	DirectDebitHold       *DirectDebitHoldDetails      `json:"direct_debit_hold,omitempty"`
}

func (c *client) GetTransactions(ctx context.Context, bankAccountId string, updatedAtFrom time.Time, pageSize int) ([]Transactions, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("v2/transactions"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("per_page", fmt.Sprint(pageSize))
	q.Add("sort_by", "updated_at:asc")
	q.Add("bank_account_id", bankAccountId)
	q.Add("updated_at_from", updatedAtFrom.Format(QONTO_TIMEFORMAT))
	req.URL.RawQuery = q.Encode()

	errorResponse := qontoErrors{}
	type qontoResponse struct {
		Transactions []Transactions `json:"transactions"`
		Meta         MetaPagination `json:"meta"`
	}
	successResponse := qontoResponse{}

	_, err = c.httpClient.Do(ctx, req, &successResponse, &errorResponse)

	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get transactions: %v", errorResponse.Error()),
			err,
		)
	}
	return successResponse.Transactions, nil
}
