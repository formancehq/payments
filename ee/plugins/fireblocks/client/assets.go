package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Asset struct {
	ID            string             `json:"id"`
	LegacyID      string             `json:"legacyId"`
	DisplayName   string             `json:"displayName"`
	DisplaySymbol string             `json:"displaySymbol"`
	BlockchainID  string             `json:"blockchainId"`
	AssetClass    string             `json:"assetClass"`
	Decimals      *int               `json:"decimals"` // populated for FIAT assets
	Onchain       *AssetOnchain      `json:"onchain"`  // populated for NATIVE/FT assets
	Metadata      *AssetSpecMetadata `json:"metadata"`
}

type AssetOnchain struct {
	Symbol    string   `json:"symbol"`
	Name      string   `json:"name"`
	Address   string   `json:"address"`
	Decimals  int      `json:"decimals"`
	Standards []string `json:"standards"`
}

type AssetSpecMetadata struct {
	Scope      string   `json:"scope"`
	Verified   bool     `json:"verified"`
	Deprecated bool     `json:"deprecated"`
	Features   []string `json:"features"`
}

// Asset classes returned by /v1/assets; we only ingest fungible/native/fiat.
const (
	AssetClassNative  = "NATIVE"
	AssetClassFT      = "FT"
	AssetClassFiat    = "FIAT"
	AssetClassNFT     = "NFT"
	AssetClassSFT     = "SFT"
	AssetClassVirtual = "VIRTUAL"
)

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
		endpoint := fmt.Sprintf("%s/v1/assets?pageSize=1000", c.baseURL)
		if cursor != "" {
			endpoint = fmt.Sprintf("%s&pageCursor=%s", endpoint, url.QueryEscape(cursor))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		var response AssetsResponse
		var errResponse fireblocksError
		_, err = c.httpClient.Do(ctx, req, &response, &errResponse)
		if err != nil {
			return nil, errResponse.wrap("failed to list assets", err)
		}

		allAssets = append(allAssets, response.Data...)

		if response.Paging.Next == "" {
			break
		}
		cursor = response.Paging.Next
	}

	return allAssets, nil
}
