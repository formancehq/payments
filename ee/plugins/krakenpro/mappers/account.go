package mappers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/models"
)

// RawBalanceToPSPAccount maps one raw Kraken BalanceEx variant (XXBT,
// XBT.M, ZUSD, ADA.S) to a PSPAccount — the coinbaseprime wallet-per-
// asset model, one account per asset class. Reference is the raw code
// (Kraken's stable id), DefaultAsset is the normalised symbol, Name is
// human-friendly ("BTC Spot"), and wallet_type metadata carries the
// class so resolveWallets can pick the spot/trading account.
//
// It is a pure builder: the caller owns the zero-balance skip + the
// always-emit-spot policy. Returns (nil, nil) only for an unsupported
// asset (not in the cached /Assets snapshot).
func RawBalanceToPSPAccount(currencies map[string]int, rawCode string, entry client.BalanceExEntry) (*models.PSPAccount, error) {
	symbol := NormalizeAsset(rawCode)
	if symbol == "" {
		return nil, fmt.Errorf("empty asset code")
	}
	if _, known := currencies[symbol]; !known {
		// Unknown asset (likely a newly-listed token not in our cached
		// /0/public/Assets snapshot). Skip rather than fail — the asset
		// cache TTL will pick it up on the next install/refresh.
		return nil, nil
	}
	ref := strings.ToUpper(strings.TrimSpace(rawCode))
	asset := FormatAsset(currencies, symbol)
	name := symbol + " " + capitalize(WalletClass(ref))
	raw, err := json.Marshal(struct {
		Code  string                `json:"code"`
		Entry client.BalanceExEntry `json:"entry"`
	}{Code: ref, Entry: entry})
	if err != nil {
		return nil, fmt.Errorf("marshal raw for %s: %w", ref, err)
	}
	return &models.PSPAccount{
		Reference:    ref,
		Name:         &name,
		CreatedAt:    KrakenGenesis,
		DefaultAsset: &asset,
		Metadata:     AccountMetadata(ref),
		Raw:          raw,
	}, nil
}

// capitalize upper-cases the first rune (strings.Title is deprecated).
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
