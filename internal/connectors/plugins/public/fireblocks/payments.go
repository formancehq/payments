package fireblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	LastUpdated int64 `json:"lastUpdated,omitempty"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	params := client.GetTransactionsParams{
		Limit:   PAGE_SIZE,
		OrderBy: "createdAt",
	}

	// Use lastUpdated as the "after" cursor for incremental sync
	if oldState.LastUpdated > 0 {
		// Add 1 millisecond to avoid fetching the same transaction
		params.After = fmt.Sprintf("%d", oldState.LastUpdated+1)
	}

	transactions, err := p.client.GetTransactions(ctx, params)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to get transactions: %w", err)
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
	var maxLastUpdated int64

	for _, tx := range transactions {
		raw, err := json.Marshal(tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to marshal transaction: %w", err)
		}

		// Track the maximum lastUpdated for the next sync
		if tx.LastUpdated > maxLastUpdated {
			maxLastUpdated = tx.LastUpdated
		}

		// Parse the amount
		var amount *big.Int
		if tx.AmountInfo.Amount != "" {
			amount, _, _ = parseAmountWithPrecision(tx.AmountInfo.Amount, tx.AssetID)
		} else if tx.Amount > 0 {
			// Fallback to the float amount
			precision := getAssetPrecision(tx.AssetID)
			multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
			amountFloat := new(big.Float).SetFloat64(tx.Amount)
			amountFloat.Mul(amountFloat, new(big.Float).SetInt(multiplier))
			amount, _ = amountFloat.Int(nil)
		}

		if amount == nil {
			amount = big.NewInt(0)
		}

		// Determine payment type
		paymentType := models.PAYMENT_TYPE_TRANSFER
		switch tx.Operation {
		case "TRANSFER", "INTERNAL_TRANSFER":
			paymentType = models.PAYMENT_TYPE_TRANSFER
		case "MINT":
			paymentType = models.PAYMENT_TYPE_PAYIN
		case "BURN":
			paymentType = models.PAYMENT_TYPE_PAYOUT
		}

		// Map status
		status := mapFireblocksStatus(tx.Status)

		// Build source and destination account references
		sourceAccount := buildAccountReference(tx.Source)
		destAccount := buildAccountReference(tx.Destination)

		metadata := map[string]string{
			"fireblocks_id":  tx.ID,
			"operation":      tx.Operation,
			"status":         tx.Status,
			"source_type":    tx.Source.Type,
			"dest_type":      tx.Destination.Type,
		}
		if tx.TxHash != "" {
			metadata["tx_hash"] = tx.TxHash
		}
		if tx.ExternalTxID != "" {
			metadata["external_tx_id"] = tx.ExternalTxID
		}
		if tx.Note != "" {
			metadata["note"] = tx.Note
		}
		if tx.SubStatus != "" {
			metadata["sub_status"] = tx.SubStatus
		}
		if tx.SourceAddress != "" {
			metadata["source_address"] = tx.SourceAddress
		}
		if tx.DestinationAddress != "" {
			metadata["destination_address"] = tx.DestinationAddress
		}

		createdAt := time.UnixMilli(tx.CreatedAt)

		payment := models.PSPPayment{
			Reference:                   tx.ID,
			CreatedAt:                   createdAt,
			Type:                        paymentType,
			Amount:                      amount,
			Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, tx.AssetID),
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      status,
			SourceAccountReference:      &sourceAccount,
			DestinationAccountReference: &destAccount,
			Metadata:                    metadata,
			Raw:                         raw,
		}

		payments = append(payments, payment)
	}

	// Update state with the latest lastUpdated timestamp
	newState := paymentsState{
		LastUpdated: maxLastUpdated,
	}
	if maxLastUpdated == 0 && oldState.LastUpdated > 0 {
		newState.LastUpdated = oldState.LastUpdated
	}

	stateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to marshal state: %w", err)
	}

	hasMore := len(transactions) == PAGE_SIZE

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: stateBytes,
		HasMore:  hasMore,
	}, nil
}

func mapFireblocksStatus(status string) models.PaymentStatus {
	switch status {
	case client.TxStatusCompleted:
		return models.PAYMENT_STATUS_SUCCEEDED
	case client.TxStatusFailed, client.TxStatusRejected, client.TxStatusBlocked:
		return models.PAYMENT_STATUS_FAILED
	case client.TxStatusCancelled:
		return models.PAYMENT_STATUS_CANCELLED
	case client.TxStatusSubmitted, client.TxStatusPendingScreening, client.TxStatusPendingAuthorization,
		client.TxStatusQueued, client.TxStatusPendingSignature, client.TxStatusPending3rdParty,
		client.TxStatusPending3rdPartyOther:
		return models.PAYMENT_STATUS_PENDING
	case client.TxStatusBroadcasting, client.TxStatusConfirming:
		return models.PAYMENT_STATUS_PENDING
	case client.TxStatusCancelling:
		return models.PAYMENT_STATUS_PENDING
	default:
		return models.PAYMENT_STATUS_PENDING
	}
}

func buildAccountReference(src client.SourceDestination) string {
	switch src.Type {
	case client.PeerTypeVaultAccount:
		return src.ID
	case client.PeerTypeExternalWallet:
		return fmt.Sprintf("external-%s", src.ID)
	case client.PeerTypeInternalWallet:
		return fmt.Sprintf("internal-%s", src.ID)
	case client.PeerTypeExchangeAccount:
		return fmt.Sprintf("exchange-%s", src.ID)
	case client.PeerTypeFiatAccount:
		return fmt.Sprintf("fiat-%s", src.ID)
	case client.PeerTypeNetworkConnection:
		return fmt.Sprintf("network-%s", src.ID)
	case client.PeerTypeOneTimeAddress:
		return fmt.Sprintf("onetime-%s", src.ID)
	default:
		if src.ID != "" {
			return src.ID
		}
		return src.Type
	}
}
