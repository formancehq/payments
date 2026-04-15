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
	m := map[string]string{
		MetadataPrefix + "bookingDate":          in.BookingDate,
		MetadataPrefix + "valueDate":            in.ValutaDate,
		MetadataPrefix + "bankTransactionCode":  in.BankTransactionCode,
		MetadataPrefix + "numberOfTransactions": fmt.Sprintf("%d", in.NumberOfTransactions),
		MetadataPrefix + "entryReference":       in.EntryReference,
		MetadataPrefix + "servicerReference":    in.ServicerReference,
		MetadataPrefix + "isReversal":           fmt.Sprintf("%t", in.IsReversal),
		MetadataPrefix + "isBatch":              fmt.Sprintf("%t", in.IsBatch),
		MetadataPrefix + "batchMessageId":       in.BatchMessageID,
		MetadataPrefix + "batchPaymentInfoId":   in.BatchPaymentInfoID,
		MetadataPrefix + "importedAt":           in.ImportedAt.UTC().Format(ImportedAtLayout),
	}

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
				RemittanceInfo *struct {
					Unstructured []string `json:"unstructured,omitempty"`
					Structured   []struct {
						CreditorReferenceInfo *struct {
							Reference string `json:"reference,omitempty"`
						} `json:"creditorReferenceInfo,omitempty"`
					} `json:"structured,omitempty"`
				} `json:"remittanceInfo,omitempty"`
			} `json:"rawDetail,omitempty"`
		}

		if err := json.Unmarshal(detail, &parsed); err != nil {
			continue
		}

		if parsed.RawDetail == nil {
			continue
		}

		if parsed.RawDetail.References != nil {
			refs := parsed.RawDetail.References
			m[MetadataPrefix+"messageId"] = refs.MessageID
			m[MetadataPrefix+"accountServicerReference"] = refs.AccountServiceReference
			m[MetadataPrefix+"paymentInformationId"] = refs.PaymentInformationID
			m[MetadataPrefix+"instructionId"] = refs.InstructionID
			m[MetadataPrefix+"endToEndId"] = refs.EndToEndID
			m[MetadataPrefix+"uetr"] = refs.UETR
			m[MetadataPrefix+"transactionId"] = refs.TransactionID
			m[MetadataPrefix+"mandateId"] = refs.MandateID
			m[MetadataPrefix+"chequeNumber"] = refs.ChequeNumber
			m[MetadataPrefix+"clearingSystemReference"] = refs.ClearingSystemReference
			m[MetadataPrefix+"accountOwnerTxId"] = refs.AccountOwnerTxID
			m[MetadataPrefix+"accountServicerTxId"] = refs.AccountServicerTxID
			m[MetadataPrefix+"marketInfrastructureTxId"] = refs.MarketInfrastructureTxID
			m[MetadataPrefix+"processingId"] = refs.ProcessingID
		}

		if parsed.RawDetail.RemittanceInfo != nil {
			remittanceInfo := parsed.RawDetail.RemittanceInfo
			if len(remittanceInfo.Unstructured) > 0 {
				m[MetadataPrefix+"remittanceInfo"] = strings.Join(remittanceInfo.Unstructured, " ")
			} else if len(remittanceInfo.Structured) > 0 && remittanceInfo.Structured[0].CreditorReferenceInfo != nil {
				m[MetadataPrefix+"remittanceInfo"] = remittanceInfo.Structured[0].CreditorReferenceInfo.Reference
			}
		}

		break // only populate metadata from the first record - other data can be found in raw
	}

	return m
}
