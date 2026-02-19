package coinbaseprime

import "github.com/formancehq/go-libs/v3/currency"

var (
	// supportedCurrenciesWithDecimal maps currency codes to their decimal precision.
	// Fiat currencies use ISO4217 standard, crypto currencies use their native precision.
	// c.f.: https://docs.cloud.coinbase.com/exchange/docs/currencies
	supportedCurrenciesWithDecimal = map[string]int{
		// Fiat currencies (from ISO4217)
		"USD": currency.ISO4217Currencies["USD"], // US Dollar (2)
		"EUR": currency.ISO4217Currencies["EUR"], // Euro (2)
		"GBP": currency.ISO4217Currencies["GBP"], // Pound Sterling (2)
		"CAD": currency.ISO4217Currencies["CAD"], // Canadian Dollar (2)
		"AUD": currency.ISO4217Currencies["AUD"], // Australian Dollar (2)
		"JPY": currency.ISO4217Currencies["JPY"], // Japanese Yen (0)
		"CHF": currency.ISO4217Currencies["CHF"], // Swiss Franc (2)
		"SGD": currency.ISO4217Currencies["SGD"], // Singapore Dollar (2)

		// Stablecoins (6 decimals as per their smart contracts)
		"USDC": 6, // USD Coin
		"USDT": 6, // Tether
		"PYUSD": 6, // PayPal USD
		"GUSD": 2, // Gemini Dollar

		// Ethereum and ERC-20 tokens (18 decimals)
		"ETH":   18, // Ethereum
		"DAI":   18, // Dai
		"WETH":  18, // Wrapped Ether
		"SHIB":  18, // Shiba Inu
		"LINK":  18, // Chainlink
		"UNI":   18, // Uniswap
		"AAVE":  18, // Aave
		"MKR":   18, // Maker
		"CRV":   18, // Curve DAO Token
		"COMP":  18, // Compound
		"SNX":   18, // Synthetix
		"GRT":   18, // The Graph
		"BAT":   18, // Basic Attention Token
		"MATIC": 18, // Polygon
		"AVAX":  18, // Avalanche
		"FET":   18, // Fetch.ai

		// Bitcoin and Bitcoin-derived (8 decimals)
		"BTC":  8, // Bitcoin
		"LTC":  8, // Litecoin
		"BCH":  8, // Bitcoin Cash
		"DOGE": 8, // Dogecoin
		"ZEC":  8, // Zcash

		// Other cryptocurrencies with specific decimals
		"XRP":  6, // Ripple (6 decimals - drops)
		"XLM":  7, // Stellar Lumens (7 decimals - stroops)
		"ALGO": 6, // Algorand
		"SOL":  9, // Solana
		"DOT":  10, // Polkadot
		"ATOM": 6, // Cosmos
		"ADA":  6, // Cardano
		"NEAR": 24, // NEAR Protocol
		"APT":  8, // Aptos
		"SUI":  9, // Sui
		"ICP":  8, // Internet Computer
		"FIL":  18, // Filecoin
		"HBAR": 8, // Hedera
		"XTZ":  6, // Tezos
		"EOS":  4, // EOS
	}
)
