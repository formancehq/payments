package moov

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

type paymentsState struct {
	StartTime time.Time `json:"start_time"`
	Skip      int       `json:"skip"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	// If start time is not set, use a default start time
	if oldState.StartTime.IsZero() {
		oldState.StartTime = time.Now().Add(-30 * 24 * time.Hour) // Default to 30 days ago
	}

	newState := paymentsState{
		StartTime: oldState.StartTime,
		Skip:      oldState.Skip,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	needMore := false
	hasMore := false

	transfers, hasMoreTransfers, err := p.client.GetTransfers(ctx, oldState.StartTime, oldState.Skip, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	for _, transfer := range transfers {
		payment, err := transferToPayment(transfer)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments = append(payments, *payment)
	}

	needMore, hasMore = pagination.ShouldFetchMore(payments, transfers, req.PageSize)
	if !needMore {
		payments = payments[:req.PageSize]
	}

	// Update state for next fetch
	if len(transfers) > 0 {
		// Update the skip for the next fetch
		newState.Skip += len(transfers)
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore || hasMoreTransfers,
	}, nil
}

// transferToPayment converts a Moov transfer to a Formance payment
func transferToPayment(transfer *moov.Transfer) (*models.PSPPayment, error) {
	raw, err := json.Marshal(transfer)
	if err != nil {
		return nil, err
	}

	// Determine payment type based on source and destination
	paymentType := models.PAYMENT_TYPE_TRANSFER
	var sourceAccountReference, destinationAccountReference *string

	// Check if source is a wallet
	if transfer.Source.WalletID != "" {
		sourceAccountReference = &transfer.Source.WalletID
	}

	// Check if destination is a wallet
	if transfer.Destination.WalletID != "" {
		destinationAccountReference = &transfer.Destination.WalletID
	}

	// Determine payment type based on source and destination
	if sourceAccountReference != nil && destinationAccountReference != nil {
		// Both source and destination are wallets, so it's a transfer
		paymentType = models.PAYMENT_TYPE_TRANSFER
	} else if sourceAccountReference != nil && destinationAccountReference == nil {
		// Source is a wallet, destination is not, so it's a payout
		paymentType = models.PAYMENT_TYPE_PAYOUT
	} else if sourceAccountReference == nil && destinationAccountReference != nil {
		// Source is not a wallet, destination is, so it's a payin
		paymentType = models.PAYMENT_TYPE_PAYIN
	}

	// Map status
	status := models.PAYMENT_STATUS_PENDING
	switch transfer.Status {
	case "created", "pending":
		status = models.PAYMENT_STATUS_PENDING
	case "completed":
		status = models.PAYMENT_STATUS_SUCCEEDED
	case "failed":
		status = models.PAYMENT_STATUS_FAILED
	case "reversed", "returned":
		status = models.PAYMENT_STATUS_CANCELLED
	default:
		status = models.PAYMENT_STATUS_OTHER
	}

	// Create metadata
	metadata := map[string]string{}
	for k, v := range transfer.Metadata {
		metadata[k] = v
	}

	// Add Moov-specific metadata
	metadata[client.MoovTransferTypeMetadataKey] = string(paymentType)
	metadata[client.MoovTransferStatusMetadataKey] = transfer.Status

	// Add source and destination type metadata
	if transfer.Source.WalletID != "" {
		metadata[client.MoovSourceTypeMetadataKey] = "wallet"
	} else if transfer.Source.BankAccountID != "" {
		metadata[client.MoovSourceTypeMetadataKey] = "bank_account"
	}

	if transfer.Destination.WalletID != "" {
		metadata[client.MoovDestinationTypeMetadataKey] = "wallet"
	} else if transfer.Destination.BankAccountID != "" {
		metadata[client.MoovDestinationTypeMetadataKey] = "bank_account"
	}

	// Create payment
	return &models.PSPPayment{
		Reference:                   transfer.ID,
		Type:                        paymentType,
		Status:                      status,
		Amount:                      big.NewInt(transfer.Amount.Value),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.Amount.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		CreatedAt:                   transfer.CreatedOn,
		SourceAccountReference:      sourceAccountReference,
		DestinationAccountReference: destinationAccountReference,
		Raw:                         raw,
		Metadata:                    metadata,
	}, nil
}