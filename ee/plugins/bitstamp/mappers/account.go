package mappers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

// AccountEnrichment is the install-time-loaded asset context folded
// into PSPAccount.Metadata. Every field is optional. List-valued
// fields are serialised with deterministic JSON ordering so the
// metadata payload is stable across cycles.
type AccountEnrichment struct {
	Networks        []client.CurrencyNetwork
	WithdrawalFees  []client.WithdrawalFee
	TradableMarkets []client.MyMarket
	MakerFee        string
	TakerFee        string
	MinOrderValue   string
	MarketType      string
}

// AccountBalanceToPSPAccount is the zero-enrichment convenience used
// by tests + the cold-install path before enrichment caches load.
func AccountBalanceToPSPAccount(currencyIndex map[string]client.Currency, bal client.AccountBalance) (*models.PSPAccount, error) {
	return AccountBalanceToPSPAccountEnriched(currencyIndex, bal, AccountEnrichment{})
}

// AccountBalanceToPSPAccountEnriched maps one /account_balances/ row
// to a PSPAccount with optional enrichment metadata. (nil, nil) on
// empty currency. Raw is preserved for the FromPayload-driven balances task.
func AccountBalanceToPSPAccountEnriched(currencyIndex map[string]client.Currency, bal client.AccountBalance, enrich AccountEnrichment) (*models.PSPAccount, error) {
	symbol := NormalizeCurrency(bal.Currency)
	if symbol == "" {
		return nil, nil
	}
	raw, err := json.Marshal(bal)
	if err != nil {
		return nil, fmt.Errorf("marshal account balance for %s: %w", symbol, err)
	}
	account := models.PSPAccount{
		Reference: symbol,
		Name:      &symbol,
		CreatedAt: BitstampGenesis,
		Raw:       raw,
	}
	if cur, known := currencyIndex[symbol]; known {
		asset := currency.FormatAssetWithPrecision(symbol, cur.Decimals)
		account.DefaultAsset = &asset
	}
	metadata, err := buildAccountMetadata(enrich)
	if err != nil {
		return nil, fmt.Errorf("build enrichment metadata for %s: %w", symbol, err)
	}
	if len(metadata) > 0 {
		account.Metadata = metadata
	}
	return &account, nil
}

// buildAccountMetadata folds an AccountEnrichment into the canonical
// metadata namespace, serialising the list-valued fields as
// deterministic JSON. Returns nil (not an empty map) when no
// enrichment fields are populated — the orchestrator can then leave
// PSPAccount.Metadata unset entirely.
func buildAccountMetadata(enrich AccountEnrichment) (map[string]string, error) {
	out := map[string]string{}

	if len(enrich.Networks) > 0 {
		sorted := append([]client.CurrencyNetwork(nil), enrich.Networks...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Network < sorted[j].Network })
		raw, err := json.Marshal(sorted)
		if err != nil {
			return nil, fmt.Errorf("marshal networks: %w", err)
		}
		out[MetadataKeyNetworks] = string(raw)
	}

	if len(enrich.WithdrawalFees) > 0 {
		sorted := append([]client.WithdrawalFee(nil), enrich.WithdrawalFees...)
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].Currency != sorted[j].Currency {
				return sorted[i].Currency < sorted[j].Currency
			}
			return sorted[i].Network < sorted[j].Network
		})
		raw, err := json.Marshal(sorted)
		if err != nil {
			return nil, fmt.Errorf("marshal withdrawal fees: %w", err)
		}
		out[MetadataKeyWithdrawalFees] = string(raw)
	}

	if len(enrich.TradableMarkets) > 0 {
		symbols := make([]string, 0, len(enrich.TradableMarkets))
		for _, m := range enrich.TradableMarkets {
			s := strings.TrimSpace(m.Name)
			if s != "" {
				symbols = append(symbols, s)
			}
		}
		if len(symbols) > 0 {
			sort.Strings(symbols)
			raw, err := json.Marshal(symbols)
			if err != nil {
				return nil, fmt.Errorf("marshal tradable markets: %w", err)
			}
			out[MetadataKeyTradableMarkets] = string(raw)
		}
	}

	if s := strings.TrimSpace(enrich.MakerFee); s != "" {
		out[MetadataKeyFeeTierMaker] = s
	}
	if s := strings.TrimSpace(enrich.TakerFee); s != "" {
		out[MetadataKeyFeeTierTaker] = s
	}
	if s := strings.TrimSpace(enrich.MinOrderValue); s != "" {
		out[MetadataKeyMinOrderValue] = s
	}
	if s := strings.TrimSpace(enrich.MarketType); s != "" {
		out[MetadataKeyMarketSymbol] = s
	}
	return out, nil
}
