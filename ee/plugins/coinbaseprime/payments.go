package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

type TransferEndpointType string

const (
	// CoinbasePrime transfer types
	transferTypeWallet          TransferEndpointType = "WALLET"
	transferTypePaymentMethod   TransferEndpointType = "PAYMENT_METHOD"
	transferTypeAddress         TransferEndpointType = "ADDRESS"
	transferTypeOther           TransferEndpointType = "OTHER"
	transferTypeMultipleAddress TransferEndpointType = "MULTIPLE_ADDRESSES"
	transferTypeCounterpartyID  TransferEndpointType = "COUNTERPARTY_ID"
)

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState incrementalState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	response, err := p.client.GetTransactions(ctx, oldState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(response.Transactions))
	for _, tx := range response.Transactions {
		payment, err := p.transactionToPayment(ctx, tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to convert transaction %s: %w", tx.ID, err)
		}
		if payment == nil {
			continue
		}
		payments = append(payments, *payment)
	}

	newState := incrementalState{Cursor: advanceCursor(oldState.Cursor, response.Pagination.NextCursor)}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  response.Pagination.HasNext,
	}, nil
}

func (p *Plugin) transactionToPayment(ctx context.Context, tx client.Transaction) (*models.PSPPayment, error) {
	if strings.EqualFold(tx.Type, TransactionTypeConversion) {
		return nil, nil
	}

	asset, precision, ok, err := p.resolveAssetAndPrecision(ctx, tx.Symbol)
	if err != nil {
		return nil, err
	}
	if !ok {
		p.logger.Infof("skipping transaction %s: unsupported currency %q", tx.ID, tx.Symbol)
		return nil, nil
	}

	raw, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}

	paymentType := transactionTypeToPaymentType(tx.Type)
	status := transactionStatusToPaymentStatus(tx.Status)

	amount, err := currency.GetAmountWithPrecisionFromString(tx.Amount, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	metadata := buildTransactionMetadata(tx)

	payment := models.PSPPayment{
		Reference: tx.ID,
		CreatedAt: tx.CreatedAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     asset,
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    status,
		Metadata:  metadata,
		Raw:       raw,
	}

	sourceAccountReference, destinationAccountReference := resolveAccountReferences(tx)
	payment.SourceAccountReference = sourceAccountReference
	payment.DestinationAccountReference = destinationAccountReference

	return &payment, nil
}

func buildTransactionMetadata(tx client.Transaction) map[string]string {
	metadata := make(map[string]string)
	set := func(k, v string) {
		if v != "" {
			metadata[MetadataPrefix+k] = v
		}
	}

	set("wallet_id", tx.WalletID)
	set("portfolio_id", tx.PortfolioID)
	metadata[MetadataPrefix+"type"] = tx.Type
	metadata[MetadataPrefix+"status"] = tx.Status

	hasFees := (tx.Fees != "" && tx.Fees != "0") || (tx.NetworkFees != "" && tx.NetworkFees != "0")
	if tx.Fees != "" && tx.Fees != "0" {
		set("fees", tx.Fees)
	}
	if hasFees {
		set("fee_symbol", tx.FeeSymbol)
	}
	if tx.NetworkFees != "" && tx.NetworkFees != "0" {
		set("network_fees", tx.NetworkFees)
	}

	set("network", tx.Network)
	if len(tx.BlockchainIDs) > 0 {
		metadata[MetadataPrefix+"blockchain_ids"] = strings.Join(tx.BlockchainIDs, ",")
	}

	if tx.CompletedAt != nil {
		metadata[MetadataPrefix+"completed_at"] = tx.CompletedAt.Format(time.RFC3339)
	}

	sourceAddress := ""
	if tx.TransferFrom != nil && tx.TransferFrom.Address != "" {
		sourceAddress = tx.TransferFrom.Address
	} else if tx.SourceAddress != "" {
		sourceAddress = tx.SourceAddress
	}
	set("source_address", sourceAddress)

	depositAddress := ""
	if tx.TransferTo != nil && tx.TransferTo.Address != "" {
		depositAddress = tx.TransferTo.Address
	} else if tx.DepositAddress != "" {
		depositAddress = tx.DepositAddress
	}
	set("deposit_address", depositAddress)
	set("external_tx_id", tx.ExternalTxID)

	return metadata
}

func isTransferType(t string, expected TransferEndpointType) bool {
	return strings.ToUpper(t) == string(expected)
}

func resolveAccountReferences(tx client.Transaction) (*string, *string) {
	var source, dest *string

	// Use transfer_from/transfer_to only when type is WALLET
	if tx.TransferFrom != nil && tx.TransferFrom.Value != "" && isTransferType(tx.TransferFrom.Type, transferTypeWallet) {
		v := tx.TransferFrom.Value
		source = &v
	}
	if tx.TransferTo != nil && tx.TransferTo.Value != "" && isTransferType(tx.TransferTo.Type, transferTypeWallet) {
		v := tx.TransferTo.Value
		dest = &v
	}

	// Fallback to walletID based on payment direction
	if tx.WalletID != "" {
		switch transactionTypeToPaymentType(tx.Type) {
		case models.PAYMENT_TYPE_PAYIN:
			if dest == nil {
				v := tx.WalletID
				dest = &v
			}
		case models.PAYMENT_TYPE_PAYOUT:
			if source == nil {
				v := tx.WalletID
				source = &v
			}
		}
	}

	return source, dest
}

func transactionTypeToPaymentType(txType string) models.PaymentType {
	switch strings.ToUpper(txType) {
	case "DEPOSIT",
		"COINBASE_DEPOSIT", "COINBASE_REFUND", "REWARD",
		"DEPOSIT_ADJUSTMENT", "CLAIM_REWARDS":
		return models.PAYMENT_TYPE_PAYIN
	case "WITHDRAWAL", "SWEEP_WITHDRAWAL",
		"PROXY_WITHDRAWAL", "BILLING_WITHDRAWAL", "WITHDRAWAL_ADJUSTMENT", "SLASH":
		return models.PAYMENT_TYPE_PAYOUT
	case "INTERNAL_DEPOSIT", "INTERNAL_WITHDRAWAL", "SWEEP_DEPOSIT", "PROXY_DEPOSIT",
		"STAKE", "RESTAKE", "PORTFOLIO_STAKE", "UNSTAKE", "PORTFOLIO_UNSTAKE":
		return models.PAYMENT_TYPE_TRANSFER
	case "KEY_REGISTRATION", "COMPLETE_UNBONDING", "WITHDRAW_UNBONDED", "STAKE_ACCOUNT_CREATE", "CHANGE_VALIDATOR",
		"DELEGATION", "UNDELEGATION", "REMOVE_AUTHORIZED_PARTY", "STAKE_AUTHORIZE_WITH_SEED", "VOTE_AUTHORIZE", "ONCHAIN_TRANSACTION":
		return models.PAYMENT_TYPE_OTHER
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}

func transactionStatusToPaymentStatus(status string) models.PaymentStatus {
	switch strings.ToUpper(status) {
	case "TRANSACTION_PENDING", "TRANSACTION_CREATED", "TRANSACTION_REQUESTED",
		"TRANSACTION_APPROVED", "TRANSACTION_GASSING", "TRANSACTION_GASSED",
		"TRANSACTION_PROVISIONED", "TRANSACTION_PLANNED", "TRANSACTION_PROCESSING",
		"TRANSACTION_RESTORED", "TRANSACTION_IMPORT_PENDING", "TRANSACTION_DELAYED",
		"TRANSACTION_BROADCASTING", "TRANSACTION_CONSTRUCTED":
		return models.PAYMENT_STATUS_PENDING
	case "TRANSACTION_DONE", "TRANSACTION_IMPORTED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "TRANSACTION_CANCELLED":
		return models.PAYMENT_STATUS_CANCELLED
	case "TRANSACTION_EXPIRED":
		return models.PAYMENT_STATUS_EXPIRED
	case "TRANSACTION_FAILED", "TRANSACTION_REJECTED":
		return models.PAYMENT_STATUS_FAILED
	case "OTHER_TRANSACTION_STATUS":
		return models.PAYMENT_STATUS_OTHER
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}
