package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Asset struct {
	ID       string        `json:"id"`
	LegacyID string        `json:"legacyId"`
	Decimals *int          `json:"decimals"` // For fiat assets (when provided)
	Onchain  *AssetOnchain `json:"onchain"`  // For crypto assets
}

type AssetOnchain struct {
	Decimals int    `json:"decimals"`
	Symbol   string `json:"symbol"`
}

type AssetsPaging struct {
	Next string `json:"next"`
}

type AssetsResponse struct {
	Data   []Asset      `json:"data"`
	Paging AssetsPaging `json:"paging"`
}

func (c *client) ListAssets(ctx context.Context) ([]Asset, error) {
	var allAssets []Asset
	var cursor string

	for {
		endpoint := fmt.Sprintf("%s/v1/assets", c.baseURL)
		if cursor != "" {
			endpoint = fmt.Sprintf("%s?pageCursor=%s", endpoint, url.QueryEscape(cursor))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		var response AssetsResponse
		var errResponse fireblocksError
		_, err = c.httpClient.Do(ctx, req, &response, &errResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to list assets: %w", err)
		}

		allAssets = append(allAssets, response.Data...)

		if response.Paging.Next == "" {
			break
		}
		cursor = response.Paging.Next
	}

	return allAssets, nil
}
