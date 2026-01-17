package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	// Coinbase Prime conversions are typically tracked via transactions
	// For now, return empty as there's no dedicated list endpoint
	// This would need to be implemented by polling transactions and filtering for conversions
	return models.FetchNextConversionsResponse{
		Conversions: []models.PSPConversion{},
		NewState:    nil,
		HasMore:     false,
	}, nil
}

func (p *Plugin) createConversion(ctx context.Context, req models.CreateConversionRequest) (models.CreateConversionResponse, error) {
	conversion := req.Conversion

	createReq := client.CreateConversionRequest{
		WalletID:     conversion.WalletID,
		SourceSymbol: conversion.SourceAsset,
		TargetSymbol: conversion.TargetAsset,
		Amount:       conversion.SourceAmount.String(),
	}

	resp, err := p.client.CreateConversion(ctx, createReq)
	if err != nil {
		return models.CreateConversionResponse{}, fmt.Errorf("failed to create conversion: %w", err)
	}

	pspConversion, err := coinbaseConversionToPSPConversion(resp.Conversion)
	if err != nil {
		return models.CreateConversionResponse{
			PollingConversionID: &resp.Conversion.ID,
		}, nil
	}

	return models.CreateConversionResponse{
		Conversion: &pspConversion,
	}, nil
}

func coinbaseConversionToPSPConversion(conv client.Conversion) (models.PSPConversion, error) {
	raw, _ := json.Marshal(conv)

	// Map status
	status := mapConversionStatus(conv.Status)

	// Parse amounts
	sourceAmount, err := parseOrderQuantity(conv.SourceAmount, conv.SourceSymbol)
	if err != nil {
		return models.PSPConversion{}, fmt.Errorf("failed to parse source amount: %w", err)
	}

	targetAmount, err := parseOrderQuantity(conv.TargetAmount, conv.TargetSymbol)
	if err != nil {
		return models.PSPConversion{}, fmt.Errorf("failed to parse target amount: %w", err)
	}

	return models.PSPConversion{
		Reference:    conv.ID,
		CreatedAt:    conv.CreatedAt,
		SourceAsset:  conv.SourceSymbol,
		TargetAsset:  conv.TargetSymbol,
		SourceAmount: sourceAmount,
		TargetAmount: targetAmount,
		WalletID:     conv.WalletID,
		Status:       status,
		Raw:          raw,
	}, nil
}

func mapConversionStatus(cbStatus string) models.ConversionStatus {
	switch strings.ToUpper(cbStatus) {
	case "PENDING":
		return models.CONVERSION_STATUS_PENDING
	case "COMPLETED":
		return models.CONVERSION_STATUS_COMPLETED
	case "FAILED":
		return models.CONVERSION_STATUS_FAILED
	default:
		return models.CONVERSION_STATUS_PENDING
	}
}
