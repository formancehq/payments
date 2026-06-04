package mappers

// testCurrencies mirrors the bitstamp test helper: a small fixed
// precision table used across every mapper test in this package.
// Keys are the *normalised* symbols (post-NormalizeAsset).
var testCurrencies = map[string]int{
	"BTC":  8,
	"ETH":  18,
	"EUR":  2,
	"USD":  2,
	"USDC": 6,
	"ADA":  8,
}
