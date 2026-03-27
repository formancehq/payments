package krakenpro

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Asset Normalization", func() {
	DescribeTable("normalizeAssetCode",
		func(input, expected string) {
			Expect(normalizeAssetCode(input)).To(Equal(expected))
		},
		// X/Z prefix stripping
		Entry("XXBT → BTC", "XXBT", "BTC"),
		Entry("XETH → ETH", "XETH", "ETH"),
		Entry("XLTC → LTC", "XLTC", "LTC"),
		Entry("XXRP → XRP", "XXRP", "XRP"),
		Entry("ZUSD → USD", "ZUSD", "USD"),
		Entry("ZEUR → EUR", "ZEUR", "EUR"),
		Entry("ZGBP → GBP", "ZGBP", "GBP"),
		Entry("ZJPY → JPY", "ZJPY", "JPY"),
		Entry("ZCAD → CAD", "ZCAD", "CAD"),

		// Suffix stripping
		Entry("ADA.S → ADA", "ADA.S", "ADA"),
		Entry("USDT.F → USDT", "USDT.F", "USDT"),
		Entry("ETH.B → ETH", "ETH.B", "ETH"),
		Entry("DOT.M → DOT", "DOT.M", "DOT"),
		Entry("SOL.T → SOL", "SOL.T", "SOL"),
		Entry("ATOM.P → ATOM", "ATOM.P", "ATOM"),

		// No transformation needed
		Entry("BTC stays BTC", "BTC", "BTC"),
		Entry("ETH stays ETH", "ETH", "ETH"),
		Entry("USDC stays USDC", "USDC", "USDC"),
		Entry("SOL stays SOL", "SOL", "SOL"),

		// XBT → BTC special case
		Entry("XBT → BTC", "XBT", "BTC"),

		// Edge cases
		Entry("empty string", "", ""),
		Entry("lowercase normalized", "xxbt", "BTC"),
		Entry("whitespace trimmed", " ZUSD ", "USD"),

		// Assets that should NOT be stripped (no X/Z prefix in known map)
		Entry("KFEE stays KFEE", "KFEE", "KFEE"),
		Entry("FLOW stays FLOW", "FLOW", "FLOW"),
	)
})
