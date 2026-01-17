package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/binance/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	// Binance doesn't have a dedicated conversions endpoint
	// Conversions are executed as market orders, so we return empty here
	// as there's no way to list past conversions separately from regular trades
	return models.FetchNextConversionsResponse{
		Conversions: []models.PSPConversion{},
		NewState:    nil,
		HasMore:     false,
	}, nil
}

func (p *Plugin) createConversion(ctx context.Context, req models.CreateConversionRequest) (models.CreateConversionResponse, error) {
	conversion := req.Conversion

	// Build symbol from source and target assets
	// For a conversion from BTC to USDT, we're selling BTC for USDT, so the symbol is "BTCUSDT"
	symbol := buildBinanceSymbol(
		stripAssetPrecision(conversion.SourceAsset),
		stripAssetPrecision(conversion.TargetAsset),
	)

	// Convert source amount to string with proper decimal formatting
	quantity := formatBigIntAsDecimal(conversion.SourceAmount, conversion.SourceAsset)

	createReq := client.CreateOrderRequest{
		Symbol:           symbol,
		Side:             "SELL", // Selling source asset to get target asset
		Type:             "MARKET",
		Quantity:         quantity,
		NewClientOrderID: conversion.Reference,
	}

	// Execute as a market sell order (converting source to target)
	// This is an instant order that executes immediately at market price
	resp, err := p.client.CreateOrder(ctx, createReq)
	if err != nil {
		return models.CreateConversionResponse{}, fmt.Errorf("failed to create conversion: %w", err)
	}

	// Parse the response to create a PSPConversion
	pspConversion, err := binanceOrderResponseToConversion(resp, conversion)
	if err != nil {
		// If we can't fully parse, return the order ID for polling
		orderID := strconv.FormatInt(resp.OrderID, 10)
		return models.CreateConversionResponse{
			PollingConversionID: &orderID,
		}, nil
	}

	return models.CreateConversionResponse{
		Conversion: &pspConversion,
	}, nil
}

func binanceOrderResponseToConversion(resp *client.CreateOrderResponse, originalConversion models.PSPConversion) (models.PSPConversion, error) {
	raw, _ := json.Marshal(resp)

	// Parse created time
	createdAt := time.UnixMilli(resp.TransactTime)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	// Parse the executed amount (cumulative quote quantity is the amount received)
	targetAmount, err := parseBinanceAmount(resp.CumulativeQuoteQty, originalConversion.TargetAsset)
	if err != nil {
		return models.PSPConversion{}, fmt.Errorf("failed to parse target amount: %w", err)
	}

	// For market orders, the status is typically completed immediately
	status := models.CONVERSION_STATUS_COMPLETED
	if strings.ToUpper(resp.Status) == "PARTIALLY_FILLED" {
		status = models.CONVERSION_STATUS_PENDING
	} else if strings.ToUpper(resp.Status) == "REJECTED" || strings.ToUpper(resp.Status) == "EXPIRED" {
		status = models.CONVERSION_STATUS_FAILED
	}

	return models.PSPConversion{
		Reference:    strconv.FormatInt(resp.OrderID, 10),
		CreatedAt:    createdAt,
		SourceAsset:  originalConversion.SourceAsset,
		TargetAsset:  originalConversion.TargetAsset,
		SourceAmount: originalConversion.SourceAmount,
		TargetAmount: targetAmount,
		WalletID:     originalConversion.WalletID,
		Status:       status,
		Raw:          raw,
	}, nil
}

// stripAssetPrecision removes the precision suffix from an asset code
// e.g., "BTC/8" -> "BTC", "USDT/6" -> "USDT"
func stripAssetPrecision(asset string) string {
	if idx := strings.Index(asset, "/"); idx != -1 {
		return asset[:idx]
	}
	return asset
}
