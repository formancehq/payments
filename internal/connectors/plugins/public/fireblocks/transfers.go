package fireblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	// Parse the amount back to a decimal string
	precision := getAssetPrecision(pi.Asset)
	amountStr := formatAmountToDecimal(pi.Amount, precision)

	// Determine source and destination based on account references
	var sourceRef, destRef string
	if pi.SourceAccount != nil {
		sourceRef = pi.SourceAccount.Reference
	}
	if pi.DestinationAccount != nil {
		destRef = pi.DestinationAccount.Reference
	}

	source := parseAccountReference(sourceRef)
	dest := parseAccountReference(destRef)

	// Build the transaction request
	txReq := client.CreateTransactionRequest{
		AssetID: extractAssetID(pi.Asset),
		Source: client.TransferPeerPath{
			Type: source.Type,
			ID:   source.ID,
		},
		Destination: client.DestinationTransferPeerPath{
			Type: dest.Type,
			ID:   dest.ID,
		},
		Amount: amountStr,
	}

	// Add note from description
	if pi.Description != "" {
		txReq.Note = pi.Description
	}

	// Add external tx ID from reference
	if pi.Reference != "" {
		txReq.ExternalTxID = pi.Reference
	}

	// Set fee level if provided in metadata
	if pi.Metadata != nil {
		if feeLevel, ok := pi.Metadata["fee_level"]; ok {
			txReq.FeeLevel = feeLevel
		}
	}

	// Determine operation type
	if source.Type == client.PeerTypeVaultAccount && dest.Type == client.PeerTypeVaultAccount {
		txReq.Operation = "INTERNAL_TRANSFER"
	} else {
		txReq.Operation = "TRANSFER"
	}

	// Create the transaction
	resp, err := p.client.CreateTransaction(ctx, txReq)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	return models.PSPPayment{
		Reference: resp.ID,
		CreatedAt: time.Now().UTC(),
		Type:      models.PAYMENT_TYPE_TRANSFER,
		Amount:    pi.Amount,
		Asset:     pi.Asset,
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_PENDING,
	}, nil
}

type accountRef struct {
	Type string
	ID   string
}

func parseAccountReference(ref string) accountRef {
	if ref == "" {
		return accountRef{Type: client.PeerTypeVaultAccount, ID: "0"}
	}

	// Parse references like "external-123", "internal-456", "123" (vault account)
	if strings.HasPrefix(ref, "external-") {
		return accountRef{
			Type: client.PeerTypeExternalWallet,
			ID:   strings.TrimPrefix(ref, "external-"),
		}
	}
	if strings.HasPrefix(ref, "internal-") {
		return accountRef{
			Type: client.PeerTypeInternalWallet,
			ID:   strings.TrimPrefix(ref, "internal-"),
		}
	}
	if strings.HasPrefix(ref, "exchange-") {
		return accountRef{
			Type: client.PeerTypeExchangeAccount,
			ID:   strings.TrimPrefix(ref, "exchange-"),
		}
	}
	if strings.HasPrefix(ref, "fiat-") {
		return accountRef{
			Type: client.PeerTypeFiatAccount,
			ID:   strings.TrimPrefix(ref, "fiat-"),
		}
	}
	if strings.HasPrefix(ref, "network-") {
		return accountRef{
			Type: client.PeerTypeNetworkConnection,
			ID:   strings.TrimPrefix(ref, "network-"),
		}
	}

	// Check if it's a vault account with asset suffix (e.g., "123-ETH")
	if parts := strings.Split(ref, "-"); len(parts) >= 1 {
		// Assume it's a vault account ID
		return accountRef{
			Type: client.PeerTypeVaultAccount,
			ID:   parts[0],
		}
	}

	// Default to vault account
	return accountRef{
		Type: client.PeerTypeVaultAccount,
		ID:   ref,
	}
}

func formatAmountToDecimal(amount *big.Int, precision int) string {
	if amount == nil || amount.Cmp(big.NewInt(0)) == 0 {
		return "0"
	}

	str := amount.String()

	// Pad with zeros if necessary
	for len(str) <= precision {
		str = "0" + str
	}

	// Insert decimal point
	insertPos := len(str) - precision
	if precision > 0 {
		str = str[:insertPos] + "." + str[insertPos:]
	}

	// Remove trailing zeros after decimal point
	if strings.Contains(str, ".") {
		str = strings.TrimRight(str, "0")
		str = strings.TrimRight(str, ".")
	}

	return str
}

func extractAssetID(asset string) string {
	// Remove any currency prefix/suffix formatting
	// e.g., "ETH/18" -> "ETH"
	if idx := strings.Index(asset, "/"); idx > 0 {
		return asset[:idx]
	}
	return asset
}

// pollTransferStatus polls for the status of a transfer
func (p *Plugin) pollTransferStatus(ctx context.Context, transferID string) (models.PSPPayment, error) {
	tx, err := p.client.GetTransaction(ctx, transferID)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to get transaction: %w", err)
	}

	status := mapFireblocksStatus(tx.Status)

	raw, _ := json.Marshal(tx)

	return models.PSPPayment{
		Reference: tx.ID,
		CreatedAt: time.UnixMilli(tx.CreatedAt),
		Type:      models.PAYMENT_TYPE_TRANSFER,
		Status:    status,
		Raw:       raw,
	}, nil
}
