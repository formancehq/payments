package fireblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
)

type balancesState struct {
	After string `json:"after,omitempty"`
}

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var oldState balancesState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	params := client.GetVaultAccountsParams{
		Limit: PAGE_SIZE,
		After: oldState.After,
	}

	resp, err := p.client.GetVaultAccounts(ctx, params)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to get vault accounts: %w", err)
	}

	balances := make([]models.PSPBalance, 0)
	now := time.Now().UTC()

	for _, vaultAccount := range resp.Accounts {
		for _, asset := range vaultAccount.Assets {
			// Parse the available balance
			available, precision, err := parseAmountWithPrecision(asset.Available, asset.ID)
			if err != nil {
				// Skip assets with parsing errors
				continue
			}

			accountRef := fmt.Sprintf("%s-%s", vaultAccount.ID, asset.ID)

			balance := models.PSPBalance{
				AccountReference: accountRef,
				Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, asset.ID),
				Amount:           available,
				CreatedAt:        now,
			}

			balances = append(balances, balance)

			// Also track frozen amount if present
			if asset.Frozen != "" && asset.Frozen != "0" {
				frozen, _, err := parseAmountWithPrecision(asset.Frozen, asset.ID)
				if err == nil && frozen.Cmp(big.NewInt(0)) > 0 {
					frozenBalance := models.PSPBalance{
						AccountReference: accountRef + "-frozen",
						Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, asset.ID),
						Amount:           frozen,
						CreatedAt:        now,
					}
					balances = append(balances, frozenBalance)
				}
			}

			// Track pending amount if present
			if asset.Pending != "" && asset.Pending != "0" {
				pending, _, err := parseAmountWithPrecision(asset.Pending, asset.ID)
				if err == nil && pending.Cmp(big.NewInt(0)) > 0 {
					pendingBalance := models.PSPBalance{
						AccountReference: accountRef + "-pending",
						Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, asset.ID),
						Amount:           pending,
						CreatedAt:        now,
					}
					balances = append(balances, pendingBalance)
				}
			}

			_ = precision // Used for formatting
		}
	}

	hasMore := resp.Paging.After != ""

	newState := balancesState{
		After: resp.Paging.After,
	}

	stateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to marshal state: %w", err)
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		NewState: stateBytes,
		HasMore:  hasMore,
	}, nil
}

// parseAmountWithPrecision parses a decimal string amount to big.Int with the appropriate precision
func parseAmountWithPrecision(amountStr string, assetID string) (*big.Int, int, error) {
	if amountStr == "" {
		return big.NewInt(0), 8, nil
	}

	precision := getAssetPrecision(assetID)

	// Split by decimal point
	parts := strings.Split(amountStr, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}

	// Pad or truncate fractional part
	if len(fracPart) > precision {
		fracPart = fracPart[:precision]
	} else {
		for len(fracPart) < precision {
			fracPart += "0"
		}
	}

	// Combine and parse
	combined := intPart + fracPart
	combined = strings.TrimLeft(combined, "0")
	if combined == "" {
		combined = "0"
	}

	result := new(big.Int)
	result.SetString(combined, 10)

	return result, precision, nil
}

// getAssetPrecision returns the decimal precision for a given asset
func getAssetPrecision(assetID string) int {
	if precision, ok := supportedCurrenciesWithDecimal[strings.ToUpper(assetID)]; ok {
		return precision
	}
	return 8 // Default precision for unknown assets
}
