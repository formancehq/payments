package bankingbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/ee/plugins/bankingbridge/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState workflowState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := workflowState{
		Cursor:             oldState.Cursor,
		LastSeenImportedAt: oldState.LastSeenImportedAt,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	pagedTrxs, hasMore, cursor, err := p.client.GetTransactions(ctx, newState.Cursor, newState.LastSeenImportedAt, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	for _, trx := range pagedTrxs {
		raw, err := json.Marshal(&trx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, ToPSPPayment(trx, raw))
		newState.LastSeenImportedAt = trx.ImportedAt.UTC().Format(ImportedAtLayout)
	}

	newState.Cursor = cursor
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func ToPSPPayment(in client.Transaction, raw json.RawMessage) models.PSPPayment {
	amount := big.NewInt(in.AmountInMinors)
	scheme, paymentType := PaymentSchemeAndType(in.BankTransactionCode)
	status := PaymentStatus(in.BankTransactionCode, in.IsReversal)

	var sourceAccount, destinationAccount *string
	if amount.Sign() < 0 { // negative value means account is being debited
		sourceAccount = pointer.For(in.AccountReference)
		amount.Abs(amount) // convert to a positive amount

		if in.BankTransactionCode == "" {
			paymentType = models.PAYMENT_TYPE_PAYOUT
		}
	} else {
		destinationAccount = pointer.For(in.AccountReference)

		if in.BankTransactionCode == "" {
			paymentType = models.PAYMENT_TYPE_PAYIN
		}
	}

	return models.PSPPayment{
		Reference:                   in.ID,
		CreatedAt:                   in.BookedAt,
		Amount:                      amount,
		Asset:                       in.Asset,
		Scheme:                      scheme,
		Type:                        paymentType,
		Status:                      status,
		SourceAccountReference:      sourceAccount,
		DestinationAccountReference: destinationAccount,
		Metadata:                    metadata(in),
		Raw:                         raw,
	}
}

func metadata(in client.Transaction) map[string]string {
	messageIds := make([]string, 0)
	accountServicerReferenceIds := make([]string, 0)
	paymentInformationIds := make([]string, 0)
	instructionIds := make([]string, 0)
	endToEndIds := make([]string, 0)
	uetrs := make([]string, 0)
	transactionIds := make([]string, 0)
	mandateIds := make([]string, 0)
	chequeNumbers := make([]string, 0)
	clearingSystemReferenceIds := make([]string, 0)
	accountOwnerTxIds := make([]string, 0)
	accountServicerTxIds := make([]string, 0)
	marketInfrastructureTxIds := make([]string, 0)
	processingIds := make([]string, 0)

	for _, detail := range in.Details {
		var parsed struct {
			RawDetail *struct {
				References *struct {
					MessageID                string `json:"messageId"`
					AccountServiceReference  string `json:"accountServicerReference"`
					PaymentInformationID     string `json:"paymentInformationId"`
					InstructionID            string `json:"instructionId"`
					EndToEndID               string `json:"endToEndId"`
					UETR                     string `json:"uetr"`
					TransactionID            string `json:"transactionId"`
					MandateID                string `json:"mandateId"`
					ChequeNumber             string `json:"chequeNumber"`
					ClearingSystemReference  string `json:"clearingSystemReference"`
					AccountOwnerTxID         string `json:"accountOwnerTxId"`
					AccountServicerTxID      string `json:"accountServicerTxId"`
					MarketInfrastructureTxID string `json:"marketInfrastructureTxId"`
					ProcessingID             string `json:"processingId"`
				} `json:"references,omitempty"`
			} `json:"rawDetail,omitempty"`
		}

		if err := json.Unmarshal(detail, &parsed); err != nil {
			continue
		}

		if parsed.RawDetail == nil {
			continue
		}

		if refs := parsed.RawDetail.References; refs != nil {
			if refs.MessageID != "" {
				messageIds = append(messageIds, refs.MessageID)
			}
			if refs.AccountServiceReference != "" {
				accountServicerReferenceIds = append(accountServicerReferenceIds, refs.AccountServiceReference)
			}
			if refs.PaymentInformationID != "" {
				paymentInformationIds = append(paymentInformationIds, refs.PaymentInformationID)
			}
			if refs.InstructionID != "" {
				instructionIds = append(instructionIds, refs.InstructionID)
			}
			if refs.EndToEndID != "" {
				endToEndIds = append(endToEndIds, refs.EndToEndID)
			}
			if refs.UETR != "" {
				uetrs = append(uetrs, refs.UETR)
			}
			if refs.TransactionID != "" {
				transactionIds = append(transactionIds, refs.TransactionID)
			}
			if refs.MandateID != "" {
				mandateIds = append(mandateIds, refs.MandateID)
			}
			if refs.ChequeNumber != "" {
				chequeNumbers = append(chequeNumbers, refs.ChequeNumber)
			}
			if refs.ClearingSystemReference != "" {
				clearingSystemReferenceIds = append(clearingSystemReferenceIds, refs.ClearingSystemReference)
			}
			if refs.AccountOwnerTxID != "" {
				accountOwnerTxIds = append(accountOwnerTxIds, refs.AccountOwnerTxID)
			}
			if refs.AccountServicerTxID != "" {
				accountServicerTxIds = append(accountServicerTxIds, refs.AccountServicerTxID)
			}
			if refs.MarketInfrastructureTxID != "" {
				marketInfrastructureTxIds = append(marketInfrastructureTxIds, refs.MarketInfrastructureTxID)
			}
			if refs.ProcessingID != "" {
				processingIds = append(processingIds, refs.ProcessingID)
			}
		}
	}

	return map[string]string{
		MetadataPrefix + "bookingDate":                 in.BookingDate,
		MetadataPrefix + "valueDate":                   in.ValutaDate,
		MetadataPrefix + "bankTransactionCode":         in.BankTransactionCode,
		MetadataPrefix + "numberOfTransactions":        fmt.Sprintf("%d", in.NumberOfTransactions),
		MetadataPrefix + "entryReference":              in.EntryReference,
		MetadataPrefix + "servicerReference":           in.ServicerReference,
		MetadataPrefix + "isReversal":                  fmt.Sprintf("%t", in.IsReversal),
		MetadataPrefix + "isBatch":                     fmt.Sprintf("%t", in.IsBatch),
		MetadataPrefix + "batchMessageId":              in.BatchMessageID,
		MetadataPrefix + "batchPaymentInfoId":          in.BatchPaymentInfoID,
		MetadataPrefix + "messageIds":                  strings.Join(messageIds, ","),
		MetadataPrefix + "accountServicerReferenceIds": strings.Join(accountServicerReferenceIds, ","),
		MetadataPrefix + "paymentInformationIds":       strings.Join(paymentInformationIds, ","),
		MetadataPrefix + "instructionIds":              strings.Join(instructionIds, ","),
		MetadataPrefix + "endToEndIds":                 strings.Join(endToEndIds, ","),
		MetadataPrefix + "uetrs":                       strings.Join(uetrs, ","),
		MetadataPrefix + "transactionIds":              strings.Join(transactionIds, ","),
		MetadataPrefix + "mandateIds":                  strings.Join(mandateIds, ","),
		MetadataPrefix + "chequeNumbers":               strings.Join(chequeNumbers, ","),
		MetadataPrefix + "clearingSystemReferenceIds":  strings.Join(clearingSystemReferenceIds, ","),
		MetadataPrefix + "accountOwnerTxIds":           strings.Join(accountOwnerTxIds, ","),
		MetadataPrefix + "accountServicerTxIds":        strings.Join(accountServicerTxIds, ","),
		MetadataPrefix + "marketInfrastructureTxIds":   strings.Join(marketInfrastructureTxIds, ","),
		MetadataPrefix + "processingIds":               strings.Join(processingIds, ","),
	}
}
