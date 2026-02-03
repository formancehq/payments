package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type VaultAccount struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	HiddenOnUI    bool         `json:"hiddenOnUI"`
	CustomerRefID string       `json:"customerRefId"`
	AutoFuel      bool         `json:"autoFuel"`
	Assets        []VaultAsset `json:"assets"`
	CreationDate  int64        `json:"creationDate"` // Unix timestamp in milliseconds
}

type VaultAsset struct {
	ID           string `json:"id"`
	Total        string `json:"total"`
	Available    string `json:"available"`
	Pending      string `json:"pending"`
	Frozen       string `json:"frozen"`
	LockedAmount string `json:"lockedAmount"`
	Staked       string `json:"staked"`
	BlockHeight  string `json:"blockHeight"`
	BlockHash    string `json:"blockHash"`
}

type VaultAccountsPagedResponse struct {
	Accounts []VaultAccount `json:"accounts"`
	Paging   Paging         `json:"paging"`
}

type Paging struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

type fireblocksError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (c *client) GetVaultAccountsPaged(ctx context.Context, cursor string, limit int) (*VaultAccountsPagedResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/vault/accounts_paged?limit=%d", c.baseURL, limit)
	if cursor != "" {
		endpoint = fmt.Sprintf("%s&after=%s", endpoint, url.QueryEscape(cursor))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var response VaultAccountsPagedResponse
	var errResponse fireblocksError
	_, err = c.httpClient.Do(ctx, req, &response, &errResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault accounts: %w", err)
	}

	return &response, nil
}

func (c *client) GetVaultAccount(ctx context.Context, vaultAccountID string) (*VaultAccount, error) {
	endpoint := fmt.Sprintf("%s/v1/vault/accounts/%s", c.baseURL, vaultAccountID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var response VaultAccount
	var errResponse fireblocksError
	_, err = c.httpClient.Do(ctx, req, &response, &errResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault account: %w", err)
	}

	return &response, nil
}

func (c *client) GetVaultAccountAsset(ctx context.Context, vaultAccountID, assetID string) (*VaultAsset, error) {
	endpoint := fmt.Sprintf("%s/v1/vault/accounts/%s/%s", c.baseURL, vaultAccountID, assetID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var response VaultAsset
	var errResponse fireblocksError
	_, err = c.httpClient.Do(ctx, req, &response, &errResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault account asset: %w", err)
	}

	return &response, nil
}
