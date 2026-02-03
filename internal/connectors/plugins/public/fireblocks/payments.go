package fireblocks

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
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

	// Fireblocks "after" expects a Unix ms timestamp and is exclusive ("created after").
	transactions, err := p.client.ListTransactions(ctx, oldState.LastCreatedAt, int(req.PageSize))
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
	assetDecimals := p.getAssetDecimals()
	newState := paymentsState{
		LastCreatedAt: oldState.LastCreatedAt,
		LastTxID:      oldState.LastTxID,
	}

	advanceState := func(tx client.Transaction) {
		if tx.CreatedAt > newState.LastCreatedAt {
			newState.LastCreatedAt = tx.CreatedAt
			newState.LastTxID = tx.ID
			return
		}
		if tx.CreatedAt == newState.LastCreatedAt {
			newState.LastTxID = tx.ID
		}
	}

	for _, tx := range transactions {
		advanceState(tx)

		// Deduplication: skip transactions we've already processed.
		// The Fireblocks API uses an "after" timestamp; we only drop the exact last
		// transaction to guard against inclusive implementations without relying on
		// ID ordering (IDs may be non-sequential).
		if tx.CreatedAt < oldState.LastCreatedAt {
			continue
		}
		if tx.CreatedAt == oldState.LastCreatedAt && tx.ID == oldState.LastTxID {
			continue
		}

		precision, err := currency.GetPrecision(assetDecimals, tx.AssetID)
		if err != nil {
			p.logger.Errorf("skipping transaction %s: unknown asset %q", tx.ID, tx.AssetID)
			continue
		}

		amount, err := currency.GetAmountWithPrecisionFromString(tx.AmountInfo.Amount, precision)
		if err != nil {
			p.logger.Errorf("skipping transaction %s: failed to parse amount %q for asset %q", tx.ID, tx.AmountInfo.Amount, tx.AssetID)
			continue
		}

		raw, err := json.Marshal(tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payment := models.PSPPayment{
			Reference: tx.ID,
			CreatedAt: time.UnixMilli(tx.CreatedAt),
			Type:      matchPaymentType(tx.Operation),
			Amount:    amount,
			Asset:     currency.FormatAsset(assetDecimals, tx.AssetID),
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    matchPaymentStatus(tx.Status),
			Raw:       raw,
		}

		if tx.Source.ID != "" {
			sourceRef := tx.Source.ID
			payment.SourceAccountReference = &sourceRef
		}

		metadata := map[string]string{}
		if len(tx.Destinations) > 0 {
			if len(tx.Destinations) == 1 && tx.Destinations[0].ID != "" {
				destRef := tx.Destinations[0].ID
				payment.DestinationAccountReference = &destRef
			} else {
				ids := make([]string, 0, len(tx.Destinations))
				for _, dest := range tx.Destinations {
					if dest.ID != "" {
						ids = append(ids, dest.ID)
					}
				}
				if len(ids) > 0 {
					metadata["destinationIds"] = strings.Join(ids, ",")
				}
			}
		} else if tx.Destination.ID != "" {
			destRef := tx.Destination.ID
			payment.DestinationAccountReference = &destRef
		}

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
