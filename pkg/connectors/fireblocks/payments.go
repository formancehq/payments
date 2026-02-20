package fireblocks

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/fireblocks/client"
)

type paymentsState struct {
	LastCreatedAt int64  `json:"lastCreatedAt"`
	LastTxID      string `json:"lastTxId"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}
	}

	// Fireblocks "after" expects a Unix ms timestamp and is exclusive ("created after").
	transactions, err := p.client.ListTransactions(ctx, oldState.LastCreatedAt, int(req.PageSize))
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	payments := make([]connector.PSPPayment, 0, len(transactions))
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
			return connector.FetchNextPaymentsResponse{}, err
		}

		payment := connector.PSPPayment{
			Reference: tx.ID,
			CreatedAt: time.UnixMilli(tx.CreatedAt),
			Type:      matchPaymentType(tx),
			Amount:    amount,
			Asset:     currency.FormatAsset(assetDecimals, tx.AssetID),
			Scheme:    connector.PAYMENT_SCHEME_OTHER,
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
		if tx.Note != "" {
			metadata["note"] = tx.Note
		}
		if tx.SubStatus != "" {
			metadata["subStatus"] = tx.SubStatus
		}
		if len(metadata) > 0 {
			payment.Metadata = metadata
		}

		payments = append(payments, payment)
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	hasMore := len(transactions) == int(req.PageSize)

	return connector.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

// isExternalPeerType returns true for peer types representing endpoints
// outside the user's Fireblocks workspace.
func isExternalPeerType(peerType string) bool {
	switch peerType {
	case "EXTERNAL_WALLET", "ONE_TIME_ADDRESS", "UNKNOWN",
		"NETWORK_CONNECTION", "FIAT_ACCOUNT", "END_USER_WALLET":
		return true
	default:
		return false
	}
}

// resolveDestinationType returns the peer type of the effective destination.
func resolveDestinationType(tx client.Transaction) string {
	if len(tx.Destinations) > 0 {
		return tx.Destinations[0].Type
	}
	return tx.Destination.Type
}

func matchPaymentType(tx client.Transaction) connector.PaymentType {
	switch tx.Operation {
	case "TRANSFER", "INTERNAL_TRANSFER":
		srcExternal := isExternalPeerType(tx.Source.Type)
		dstExternal := isExternalPeerType(resolveDestinationType(tx))
		switch {
		case srcExternal && !dstExternal:
			return connector.PAYMENT_TYPE_PAYIN
		case !srcExternal && dstExternal:
			return connector.PAYMENT_TYPE_PAYOUT
		default:
			return connector.PAYMENT_TYPE_TRANSFER
		}
	default:
		return connector.PAYMENT_TYPE_OTHER
	}
}

func matchPaymentStatus(status string) connector.PaymentStatus {
	switch status {
	case "COMPLETED":
		return connector.PAYMENT_STATUS_SUCCEEDED
	case "SUBMITTED", "PENDING_AML_SCREENING", "PENDING_ENRICHMENT",
		"PENDING_AUTHORIZATION", "QUEUED", "PENDING_SIGNATURE",
		"PENDING_3RD_PARTY_MANUAL_APPROVAL", "PENDING_3RD_PARTY",
		"CONFIRMING", "BROADCASTING":
		return connector.PAYMENT_STATUS_PENDING
	case "CANCELLING", "CANCELLED":
		return connector.PAYMENT_STATUS_CANCELLED
	case "FAILED", "BLOCKED", "REJECTED":
		return connector.PAYMENT_STATUS_FAILED
	default:
		return connector.PAYMENT_STATUS_OTHER
	}
}
