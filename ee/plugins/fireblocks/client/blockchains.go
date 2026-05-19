package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Blockchain mirrors the subset of /v1/blockchains we need to detect testnet
// assets via the authoritative `onchain.test` boolean.
type Blockchain struct {
	ID            string              `json:"id"`
	LegacyID      string              `json:"legacyId"`
	DisplayName   string              `json:"displayName"`
	NativeAssetID string              `json:"nativeAssetId"`
	Onchain       *BlockchainOnchain  `json:"onchain"`
	Metadata      *BlockchainMetadata `json:"metadata"`
}

type BlockchainOnchain struct {
	Protocol    string `json:"protocol"`
	ChainID     string `json:"chainId"`
	Test        bool   `json:"test"`
	SigningAlgo string `json:"signingAlgo"`
}

type BlockchainMetadata struct {
	Scope      string `json:"scope"`
	Deprecated bool   `json:"deprecated"`
}

type BlockchainsResponse struct {
	Data []Blockchain `json:"data"`
	Next string       `json:"next"`
}

func (c *client) ListBlockchains(ctx context.Context) ([]Blockchain, error) {
	var all []Blockchain
	var cursor string

	for {
		endpoint := fmt.Sprintf("%s/v1/blockchains?pageSize=500", c.baseURL)
		if cursor != "" {
			endpoint = fmt.Sprintf("%s&pageCursor=%s", endpoint, url.QueryEscape(cursor))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		var resp BlockchainsResponse
		var errResp fireblocksError
		if _, err := c.httpClient.Do(ctx, req, &resp, &errResp); err != nil {
			return nil, errResp.wrap("failed to list blockchains", err)
		}

		all = append(all, resp.Data...)
		if resp.Next == "" {
			break
		}
		cursor = resp.Next
	}

	return all, nil
}
