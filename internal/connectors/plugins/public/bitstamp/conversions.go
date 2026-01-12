package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

type conversionsState struct {
	LastSync time.Time `json:"last_sync"`
}

func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	// Bitstamp doesn't have a dedicated conversions endpoint
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

	// Build market pair from source and target assets
	// For a conversion from BTC to USD, we're selling BTC for USD, so the market is "btcusd"
	market := buildBitstampMarket(
		stripAssetPrecision(conversion.SourceAsset),
		stripAssetPrecision(conversion.TargetAsset),
	)

	// Convert source amount to string with proper decimal formatting
	amount := formatBigIntAsDecimal(conversion.SourceAmount, conversion.SourceAsset)

	createReq := client.CreateOrderRequest{
		Market:        market,
		Amount:        amount,
		ClientOrderID: conversion.Reference,
	}

	// Execute as a market sell order (converting source to target)
	// This is an instant order that executes immediately at market price
	resp, err := p.client.CreateMarketSellOrder(ctx, createReq)
	if err != nil {
		return models.CreateConversionResponse{}, fmt.Errorf("failed to create conversion: %w", err)
	}

	// Parse the response to create a PSPConversion
	pspConversion, err := bitstampOrderResponseToConversion(resp, conversion)
	if err != nil {
		// If we can't fully parse, return the order ID for polling
		return models.CreateConversionResponse{
			PollingConversionID: &resp.ID,
		}, nil
	}

	return models.CreateConversionResponse{
		Conversion: &pspConversion,
	}, nil
}

func bitstampOrderResponseToConversion(resp *client.CreateOrderResponse, originalConversion models.PSPConversion) (models.PSPConversion, error) {
	raw, _ := json.Marshal(resp)

	// Parse created time
	createdAt, err := time.Parse("2006-01-02 15:04:05", resp.DateTime)
	if err != nil {
		createdAt = time.Now()
	}

	// Parse the executed amount
	targetAmount, err := parseBitstampAmount(resp.Price, originalConversion.TargetAsset)
	if err != nil {
		return models.PSPConversion{}, fmt.Errorf("failed to parse target amount: %w", err)
	}

	// For instant orders, the status is typically completed immediately
	status := models.CONVERSION_STATUS_COMPLETED

	return models.PSPConversion{
		Reference:    resp.ID,
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
// e.g., "BTC/8" -> "BTC", "USD/2" -> "USD"
func stripAssetPrecision(asset string) string {
	if idx := strings.Index(asset, "/"); idx != -1 {
		return asset[:idx]
	}
	return asset
}
