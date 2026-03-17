package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/models"
)

// ledgerPageSize matches PAGE_SIZE from config.go — Kraken returns max 50 entries per Ledgers call.
const ledgerPageSize = PAGE_SIZE

type paymentsState struct {
	Offset    int   `json:"offset"`
	LastSeenTime int64 `json:"lastSeenTime,omitempty"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	// Use `start` to fetch entries newer than the last seen timestamp.
	// On the first call, LastSeenTime is 0 which fetches from the beginning.
	response, err := p.client.GetLedgers(ctx, oldState.Offset, oldState.LastSeenTime)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	// Sort ledger entries by key for deterministic output
	ledgerIDs := make([]string, 0, len(response.Result.Ledgers))
	for id := range response.Result.Ledgers {
		ledgerIDs = append(ledgerIDs, id)
	}
	sort.Strings(ledgerIDs)

	payments := make([]models.PSPPayment, 0, len(response.Result.Ledgers))
	for _, ledgerID := range ledgerIDs {
		entry := response.Result.Ledgers[ledgerID]

		payment, err := p.ledgerEntryToPayment(ledgerID, entry)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to convert ledger %s: %w", ledgerID, err)
		}
		if payment == nil {
			continue
		}
		payments = append(payments, *payment)
	}

	hasMore := len(response.Result.Ledgers) >= ledgerPageSize

	// Track the latest timestamp seen for the next polling cycle
	latestTime := oldState.LastSeenTime
	for _, entry := range response.Result.Ledgers {
		entryTime := int64(entry.Time)
		if entryTime > latestTime {
			latestTime = entryTime
		}
	}

	newState := paymentsState{
		Offset:       oldState.Offset + len(response.Result.Ledgers),
		LastSeenTime: latestTime,
	}

	// When there are no more pages, reset offset for the next polling cycle
	// so future runs fetch entries newer than the latest seen.
	if !hasMore {
		newState.Offset = 0
	}
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

func (p *Plugin) ledgerEntryToPayment(ledgerID string, entry client.LedgerEntry) (*models.PSPPayment, error) {
	normalized := normalizeAssetCode(entry.Asset)
	if normalized == "" {
		p.logger.Infof("skipping ledger %s: empty asset after normalization", ledgerID)
		return nil, nil
	}

	precision := p.getPrecision(normalized)

	// Parse amount (can be negative for debits)
	amountStr := strings.TrimPrefix(entry.Amount, "-")

	amount, err := currency.GetAmountWithPrecisionFromString(truncateToPrecision(amountStr, precision), precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	asset := currency.FormatAsset(p.formattedCurrMap, normalized)

	raw, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}

	createdAt := time.Unix(int64(entry.Time), 0).UTC()

	payment := models.PSPPayment{
		Reference: ledgerID,
		CreatedAt: createdAt,
		Type:      ledgerTypeToPaymentType(entry.Type),
		Amount:    amount,
		Asset:     asset,
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       raw,
		Metadata:  buildLedgerMetadata(ledgerID, entry),
	}

	// For deposits, the account is the destination
	// For withdrawals/spend/sale, the account is the source
	accountRef := p.accountRef
	switch payment.Type {
	case models.PAYMENT_TYPE_PAYIN:
		payment.DestinationAccountReference = &accountRef
	case models.PAYMENT_TYPE_PAYOUT:
		payment.SourceAccountReference = &accountRef
	case models.PAYMENT_TYPE_TRANSFER:
		payment.SourceAccountReference = &accountRef
		payment.DestinationAccountReference = &accountRef
	}

	return &payment, nil
}

func buildLedgerMetadata(ledgerID string, entry client.LedgerEntry) map[string]string {
	metadata := map[string]string{
		"ledger_id":   ledgerID,
		"type":        entry.Type,
		"raw_asset":   entry.Asset,
		"raw_amount":  entry.Amount,
	}

	if entry.RefID != "" {
		metadata["refid"] = entry.RefID
	}
	if entry.Subtype != "" {
		metadata["subtype"] = entry.Subtype
	}
	if entry.Fee != "" {
		if f, ok := new(big.Float).SetString(entry.Fee); ok && f.Sign() != 0 {
			metadata["fee"] = entry.Fee
		}
	}
	if entry.Balance != "" {
		metadata["post_balance"] = entry.Balance
	}

	return metadata
}

func ledgerTypeToPaymentType(ledgerType string) models.PaymentType {
	switch strings.ToLower(strings.TrimSpace(ledgerType)) {
	case "deposit":
		return models.PAYMENT_TYPE_PAYIN
	case "withdrawal":
		return models.PAYMENT_TYPE_PAYOUT
	case "trade":
		return models.PAYMENT_TYPE_TRANSFER
	case "transfer":
		return models.PAYMENT_TYPE_TRANSFER
	case "margin":
		return models.PAYMENT_TYPE_OTHER
	case "rollover":
		return models.PAYMENT_TYPE_OTHER
	case "spend":
		return models.PAYMENT_TYPE_PAYOUT
	case "receive":
		return models.PAYMENT_TYPE_PAYIN
	case "settled":
		return models.PAYMENT_TYPE_OTHER
	case "adjustment":
		return models.PAYMENT_TYPE_OTHER
	case "staking":
		return models.PAYMENT_TYPE_PAYIN
	case "sale":
		return models.PAYMENT_TYPE_PAYOUT
	case "dividend":
		return models.PAYMENT_TYPE_PAYIN
	case "nft_trade":
		return models.PAYMENT_TYPE_TRANSFER
	case "nft_rebate":
		return models.PAYMENT_TYPE_PAYIN
	case "credit":
		return models.PAYMENT_TYPE_PAYIN
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}
