package mappers

import (
	"strings"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
)

// WalletClassSpot is the trading class: the suffix-free spot wallet
// orders + conversions debit/credit. It is the `wallet_type` value
// resolveWallets filters on, mirroring coinbaseprime's TRADING filter.
const WalletClassSpot = "spot"

// WalletClass maps a raw Kraken code to its asset-class label, the
// `wallet_type` discriminator each PSPAccount carries (coinbaseprime
// pattern). Spot (suffix-free) is the trading class; the staking/earn
// suffix families each get their own class so balances stay separable.
func WalletClass(rawCode string) string {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	for _, suffix := range suffixFamilies {
		if strings.HasSuffix(code, suffix) {
			return suffixClassLabel(suffix)
		}
	}
	return WalletClassSpot
}

// suffixClassLabel maps a documented suffix family to its class label.
func suffixClassLabel(suffix string) string {
	switch suffix {
	case ".S":
		return "staked"
	case ".M":
		return "rewards"
	case ".B":
		return "yield"
	case ".F":
		return "earn"
	case ".P":
		return "parachain"
	case ".T":
		return "tokenised"
	case ".HOLD":
		return "hold"
	case ".BASE":
		return "margin"
	default:
		return WalletClassSpot
	}
}

// LedgerMetadata builds the per-PSPPayment / per-PSPConversion
// metadata bundle, namespaced via MetadataPrefix. kraken_asset carries
// the raw Kraken asset code (e.g. "XXBT", "XBT.M") so the original
// spot/earn provenance stays queryable even though the PSPPayment's
// Asset field is normalised. Empty / zero optional fields are omitted
// so downstream filtering by presence stays meaningful.
func LedgerMetadata(ledgerID string, e client.LedgerEntry) map[string]string {
	m := map[string]string{
		MetadataPrefix + "ledger_id":    ledgerID,
		MetadataPrefix + "refid":        e.Refid,
		MetadataPrefix + "kraken_type":  e.Type,
		MetadataPrefix + "kraken_asset": e.Asset,
		MetadataPrefix + "aclass":       e.Aclass,
	}
	if e.Subtype != "" {
		m[MetadataPrefix+"subtype"] = e.Subtype
	}
	if e.Fee != "" && !IsZeroAmount(e.Fee) {
		m[MetadataPrefix+"fee"] = e.Fee
	}
	if e.Balance != "" {
		m[MetadataPrefix+"balance_after"] = e.Balance
	}
	return m
}

// AccountMetadata is the per-PSPAccount metadata bundle for one raw
// Kraken variant. wallet_type is the class discriminator resolveWallets
// keys off (coinbaseprime pattern); kraken_asset is the raw code, which
// equals the account Reference and keeps the spot/earn provenance
// explicit alongside the normalised DefaultAsset.
func AccountMetadata(rawCode string) map[string]string {
	return map[string]string{
		MetadataPrefix + "wallet_type":  WalletClass(rawCode),
		MetadataPrefix + "kraken_asset": strings.ToUpper(strings.TrimSpace(rawCode)),
	}
}

// OrderMetadata builds the per-PSPOrder metadata bundle. fills carries
// the per-fill txids verbatim so fill-level traceability survives the
// cumulative-state emission model (see MAPPINGS §8).
func OrderMetadata(pair, wsname string, fills []string, ordertype, priceAsset string) map[string]string {
	m := map[string]string{
		MetadataPrefix + "pair":        pair,
		MetadataPrefix + "ws_name":     wsname,
		MetadataPrefix + "ordertype":   ordertype,
		MetadataPrefix + "price_asset": priceAsset,
	}
	if len(fills) > 0 {
		m[MetadataPrefix+"fills"] = strings.Join(fills, ",")
	}
	return m
}
