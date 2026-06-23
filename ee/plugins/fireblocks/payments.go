package fireblocks

import (
	"context"
	"encoding/json"
	"maps"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/ee/plugins/fireblocks/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

type PeerType string

const (
	// Fireblocks peer types - internal vs external
	peerTypeVaultAccount   PeerType = "VAULT_ACCOUNT"
	peerTypeExternalWallet PeerType = "EXTERNAL_WALLET"
	peerTypeOneTimeAddress PeerType = "ONE_TIME_ADDRESS"
	peerTypeNetworkConn    PeerType = "NETWORK_CONNECTION"
	peerTypeFiatAccount    PeerType = "FIAT_ACCOUNT"
	peerTypeEndUserWallet  PeerType = "END_USER_WALLET"
	peerTypeUnknown        PeerType = "UNKNOWN"
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

		info, ok := p.lookupAsset(tx.AssetID)
		if !ok {
			p.logger.Errorf("skipping transaction %s: unknown asset %q", tx.ID, tx.AssetID)
			continue
		}

		amount, err := currency.GetAmountWithPrecisionFromString(tx.AmountInfo.Amount, info.Precision)
		if err != nil {
			p.logger.Errorf("skipping transaction %s: unparseable amount %q for asset %q",
				tx.ID, tx.AmountInfo.Amount, tx.AssetID)
			continue
		}

		raw, err := json.Marshal(tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payment := models.PSPPayment{
			Reference: tx.ID,
			CreatedAt: time.UnixMilli(tx.CreatedAt),
			Type:      matchPaymentType(tx),
			Amount:    amount,
			Asset:     info.Asset,
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    matchPaymentStatus(tx.Status),
			Raw:       raw,
			Metadata:  buildPaymentMetadata(tx, info),
		}

		if tx.Source.ID != "" && isPeerType(tx.Source.Type, peerTypeVaultAccount) {
			sourceRef := tx.Source.ID
			payment.SourceAccountReference = &sourceRef
		}

		if len(tx.Destinations) > 0 {
			if len(tx.Destinations) == 1 && tx.Destinations[0].ID != "" {
				if isPeerType(tx.Destinations[0].Type, peerTypeVaultAccount) {
					destRef := tx.Destinations[0].ID
					payment.DestinationAccountReference = &destRef
				}
			} else {
				ids := make([]string, 0, len(tx.Destinations))
				for _, dest := range tx.Destinations {
					if dest.ID != "" {
						ids = append(ids, dest.ID)
					}
				}
				if len(ids) > 0 {
					payment.Metadata[MetadataPrefix+"destination_ids"] = strings.Join(ids, ",")
				}
			}
		} else if tx.Destination.ID != "" && isPeerType(tx.Destination.Type, peerTypeVaultAccount) {
			destRef := tx.Destination.ID
			payment.DestinationAccountReference = &destRef
		}

		if err := payment.Validate(); err != nil {
			p.logger.Infof("dropping invalid payment %s: %s", tx.ID, err)
			continue
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

// buildPaymentMetadata seeds the payment metadata with the per-asset slice
// from the cache and layers tx-level details under MetadataPrefix. Always
// returns a non-nil map so downstream writes can happen unconditionally.
func buildPaymentMetadata(tx client.Transaction, info assetInfo) map[string]string {
	out := make(map[string]string, len(info.Metadata)+4)
	maps.Copy(out, info.Metadata)
	if tx.TxHash != "" {
		out[MetadataPrefix+"tx_hash"] = tx.TxHash
	}
	if tx.FeeInfo.NetworkFee != "" {
		out[MetadataPrefix+"network_fee"] = tx.FeeInfo.NetworkFee
	}
	if tx.Note != "" {
		out[MetadataPrefix+"note"] = tx.Note
	}
	if tx.SubStatus != "" {
		out[MetadataPrefix+"sub_status"] = tx.SubStatus
	}
	return out
}

// isPeerType checks if a peer type string matches the expected PeerType (case-insensitive).
func isPeerType(t string, expected PeerType) bool {
	return strings.ToUpper(t) == string(expected)
}

// isExternalPeerType returns true for peer types representing endpoints
// outside the user's Fireblocks workspace.
func isExternalPeerType(peerType string) bool {
	switch strings.ToUpper(peerType) {
	case string(peerTypeExternalWallet), string(peerTypeOneTimeAddress), string(peerTypeUnknown),
		string(peerTypeNetworkConn), string(peerTypeFiatAccount), string(peerTypeEndUserWallet):
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

func matchPaymentType(tx client.Transaction) models.PaymentType {
	switch tx.Operation {
	case "TRANSFER", "INTERNAL_TRANSFER":
		srcExternal := isExternalPeerType(tx.Source.Type)
		dstExternal := isExternalPeerType(resolveDestinationType(tx))
		switch {
		case srcExternal && !dstExternal:
			return models.PAYMENT_TYPE_PAYIN
		case !srcExternal && dstExternal:
			return models.PAYMENT_TYPE_PAYOUT
		default:
			return models.PAYMENT_TYPE_TRANSFER
		}
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}

func matchPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "COMPLETED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "SUBMITTED", "PENDING_AML_SCREENING", "PENDING_ENRICHMENT",
		"PENDING_AUTHORIZATION", "QUEUED", "PENDING_SIGNATURE",
		"PENDING_3RD_PARTY_MANUAL_APPROVAL", "PENDING_3RD_PARTY",
		"CONFIRMING", "BROADCASTING":
		return models.PAYMENT_STATUS_PENDING
	case "CANCELLING", "CANCELLED":
		return models.PAYMENT_STATUS_CANCELLED
	case "FAILED", "BLOCKED", "REJECTED":
		return models.PAYMENT_STATUS_FAILED
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}
