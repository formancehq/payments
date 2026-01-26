package kraken

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	// Kraken doesn't have a dedicated conversions endpoint
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

	// Build pair from source and target assets
	// For a conversion from BTC to USD, we're selling BTC for USD
	pair := buildKrakenPair(
		stripAssetPrecision(conversion.SourceAsset),
		stripAssetPrecision(conversion.TargetAsset),
	)

	// Convert source amount to string
	volume := conversion.SourceAmount.String()

	createReq := client.CreateOrderRequest{
		OrderType:     "market",
		Type:          "sell", // Selling source asset to get target asset
		Volume:        volume,
		Pair:          pair,
		ClientOrderID: conversion.Reference,
	}

	// Execute as a market sell order (converting source to target)
	// This is an instant order that executes immediately at market price
	resp, err := p.client.CreateOrder(ctx, createReq)
	if err != nil {
		return models.CreateConversionResponse{}, fmt.Errorf("failed to create conversion: %w", err)
	}

	// Parse the response to create a PSPConversion
	if len(resp.TxID) > 0 {
		orderID := resp.TxID[0]

		pspConversion := krakenOrderResponseToConversion(orderID, conversion)

		return models.CreateConversionResponse{
			Conversion: &pspConversion,
		}, nil
	}

	return models.CreateConversionResponse{}, fmt.Errorf("no transaction ID returned from Kraken")
}

func krakenOrderResponseToConversion(orderID string, originalConversion models.PSPConversion) models.PSPConversion {
	raw, _ := json.Marshal(map[string]string{"txid": orderID})

	// For market orders on Kraken, we don't get the executed price immediately
	// The order is submitted and will execute at market price
	// Status is COMPLETED since market orders execute immediately
	return models.PSPConversion{
		Reference:    orderID,
		CreatedAt:    time.Now(),
		SourceAsset:  originalConversion.SourceAsset,
		TargetAsset:  originalConversion.TargetAsset,
		SourceAmount: originalConversion.SourceAmount,
		TargetAmount: nil, // Will be filled when order completes
		WalletID:     originalConversion.WalletID,
		Status:       models.CONVERSION_STATUS_PENDING, // Market orders need confirmation
		Raw:          raw,
	}
}

// stripAssetPrecision removes the precision suffix from an asset code
// e.g., "BTC/8" -> "BTC", "USD/2" -> "USD"
func stripAssetPrecision(asset string) string {
	if idx := strings.Index(asset, "/"); idx != -1 {
		return asset[:idx]
	}
	return asset
}
