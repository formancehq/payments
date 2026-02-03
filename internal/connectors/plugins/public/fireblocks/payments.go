package fireblocks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	LastCreatedAt int64  `json:"lastCreatedAt"`
	LastTxID      string `json:"lastTxId"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	transactions, err := p.client.ListTransactions(ctx, oldState.LastCreatedAt, int(req.PageSize))
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
	newState := paymentsState{
		LastCreatedAt: oldState.LastCreatedAt,
		LastTxID:      oldState.LastTxID,
	}

	for _, tx := range transactions {
		// Deduplication: skip transactions we've already processed.
		// We use ID comparison as a tiebreaker when timestamps match.
		// Note: This assumes IDs are comparable strings; in the rare case of
		// duplicate timestamps, we may reprocess a transaction (which is safe
		// as processing is idempotent) rather than miss one.
		if tx.CreatedAt < oldState.LastCreatedAt {
			continue
		}
		if tx.CreatedAt == oldState.LastCreatedAt && tx.ID <= oldState.LastTxID {
			continue
		}

		precision, err := currency.GetPrecision(p.assetDecimals, tx.AssetID)
		if err != nil {
			p.logger.Infof("skipping transaction %s: unknown asset %q", tx.ID, tx.AssetID)
			continue
		}

		amount, err := currency.GetAmountWithPrecisionFromString(tx.AmountInfo.Amount, precision)
		if err != nil {
			p.logger.Infof("skipping transaction %s: failed to parse amount %q for asset %q", tx.ID, tx.AmountInfo.Amount, tx.AssetID)
			continue
		}

		raw, err := json.Marshal(tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payment := models.PSPPayment{
			Reference: tx.ID,
			CreatedAt: time.Unix(tx.CreatedAt/1000, (tx.CreatedAt%1000)*int64(time.Millisecond)),
			Type:      matchPaymentType(tx.Operation),
			Amount:    amount,
			Asset:     currency.FormatAsset(p.assetDecimals, tx.AssetID),
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    matchPaymentStatus(tx.Status),
			Raw:       raw,
		}

		if tx.Source.ID != "" {
			sourceRef := tx.Source.ID
			payment.SourceAccountReference = &sourceRef
		}
		if tx.Destination.ID != "" {
			destRef := tx.Destination.ID
			payment.DestinationAccountReference = &destRef
		}

		metadata := map[string]string{}
		if tx.TxHash != "" {
			metadata["txHash"] = tx.TxHash
		}
		if tx.FeeInfo.NetworkFee != "" {
			metadata["networkFee"] = tx.FeeInfo.NetworkFee
		}
		if len(metadata) > 0 {
			payment.Metadata = metadata
		}

		payments = append(payments, payment)
		newState.LastCreatedAt = tx.CreatedAt
		newState.LastTxID = tx.ID
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	hasMore := len(transactions) == int(req.PageSize)

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func matchPaymentType(operation string) models.PaymentType {
	switch operation {
	case "TRANSFER", "INTERNAL_TRANSFER":
		return models.PAYMENT_TYPE_TRANSFER
	case "DEPOSIT":
		return models.PAYMENT_TYPE_PAYIN
	case "WITHDRAW":
		return models.PAYMENT_TYPE_PAYOUT
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}

func matchPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "COMPLETED", "CONFIRMING", "CONFIRMED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "SUBMITTED", "QUEUED", "PENDING_SIGNATURE", "PENDING_AUTHORIZATION",
		"PENDING_3RD_PARTY_MANUAL_APPROVAL", "PENDING_3RD_PARTY",
		"PENDING_AML_SCREENING", "BROADCASTING":
		return models.PAYMENT_STATUS_PENDING
	case "FAILED", "REJECTED", "TIMEOUT", "CANCELLED":
		return models.PAYMENT_STATUS_FAILED
	case "BLOCKED":
		return models.PAYMENT_STATUS_CANCELLED
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}
