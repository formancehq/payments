package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	var oldState incrementalState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextConversionsResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	resp, err := p.client.GetTransactions(ctx, oldState.Cursor, req.PageSize, TransactionTypeConversion)
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("failed to list transactions: %w", err)
	}

	conversions := make([]models.PSPConversion, 0, len(resp.Transactions))
	for _, tx := range resp.Transactions {
		conv, err := p.transactionToConversion(tx)
		if err != nil {
			return models.FetchNextConversionsResponse{}, fmt.Errorf("failed to convert transaction %s: %w", tx.ID, err)
		}
		if conv != nil {
			conversions = append(conversions, *conv)
		}
	}

	newState := incrementalState{Cursor: advanceCursor(oldState.Cursor, resp.Pagination.NextCursor)}
	stateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("failed to marshal state: %w", err)
	}

	return models.FetchNextConversionsResponse{
		Conversions: conversions,
		NewState:    stateBytes,
		HasMore:     resp.Pagination.HasNext,
	}, nil
}

func (p *Plugin) transactionToConversion(tx client.Transaction) (*models.PSPConversion, error) {
	sourceAsset, sourcePrecision, sourceOk := p.resolveAssetAndPrecision(tx.Symbol)
	if !sourceOk {
		p.logger.Infof("skipping conversion %s: unsupported source currency %q", tx.ID, tx.Symbol)
		return nil, nil
	}

	targetSymbol := tx.DestinationSymbol
	if targetSymbol == "" {
		p.logger.Infof("skipping conversion %s: missing destination_symbol", tx.ID)
		return nil, nil
	}

	targetAsset, targetPrecision, targetOk := p.resolveAssetAndPrecision(targetSymbol)
	if !targetOk {
		p.logger.Infof("skipping conversion %s: unsupported target currency %q", tx.ID, targetSymbol)
		return nil, nil
	}

	raw, err := json.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw: %w", err)
	}

	sourceAmount, err := currency.GetAmountWithPrecisionFromString(tx.Amount, sourcePrecision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source amount: %w", err)
	}

	// For 1:1 conversions (e.g. stablecoin USDC->USD), the Coinbase API provides
	// a single amount field. Source and destination represent the same nominal value
	// parsed at each asset's precision. For FX conversions where amounts truly differ,
	// the PSP plugin must supply distinct values from the API response.
	targetAmount, err := currency.GetAmountWithPrecisionFromString(tx.Amount, targetPrecision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target amount: %w", err)
	}

	var fee *big.Int
	var feeAsset *string
	if tx.Fees != "" && tx.Fees != "0" {
		feeSymbol := tx.FeeSymbol
		if feeSymbol == "" {
			feeSymbol = tx.Symbol
		}
		fAsset, fPrecision, fOk := p.resolveAssetAndPrecision(feeSymbol)
		if !fOk {
			p.logger.Infof("skipping fee for conversion %s: unsupported fee currency %q", tx.ID, feeSymbol)
		} else {
			fee, err = currency.GetAmountWithPrecisionFromString(tx.Fees, fPrecision)
			if err != nil {
				return nil, fmt.Errorf("failed to parse fee: %w", err)
			}
			feeAsset = &fAsset
		}
	}

	status := mapTransactionToConversionStatus(tx.Status)

	var sourceWalletID, targetWalletID string
	if tx.TransferFrom != nil && tx.TransferFrom.Value != "" {
		sourceWalletID = tx.TransferFrom.Value
	}
	if tx.TransferTo != nil && tx.TransferTo.Value != "" {
		targetWalletID = tx.TransferTo.Value
	}

	metadata := map[string]string{
		MetadataPrefix + "transaction_id": tx.TransactionID,
		MetadataPrefix + "type":           tx.Type,
	}
	if tx.PortfolioID != "" {
		metadata[MetadataPrefix+"portfolio_id"] = tx.PortfolioID
	}

	return &models.PSPConversion{
		Reference:      tx.ID,
		CreatedAt:      tx.CreatedAt,
		SourceAsset:    sourceAsset,
		DestinationAsset:    targetAsset,
		SourceAmount:   sourceAmount,
		DestinationAmount:   targetAmount,
		Fee:            fee,
		FeeAsset:       feeAsset,
		Status:                      status,
		SourceAccountReference:      ptrIfNotEmpty(sourceWalletID),
		DestinationAccountReference: ptrIfNotEmpty(targetWalletID),
		Metadata:                    metadata,
		Raw:                         raw,
	}, nil
}

func ptrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func mapTransactionToConversionStatus(status string) models.ConversionStatus {
	switch strings.ToUpper(status) {
	case "TRANSACTION_DONE", "TRANSACTION_IMPORTED":
		return models.CONVERSION_STATUS_COMPLETED
	case "TRANSACTION_FAILED", "TRANSACTION_REJECTED", "TRANSACTION_CANCELLED":
		return models.CONVERSION_STATUS_FAILED
	default:
		return models.CONVERSION_STATUS_PENDING
	}
}
