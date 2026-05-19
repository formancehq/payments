package fireblocks

import (
	"github.com/formancehq/payments/ee/plugins/fireblocks/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fireblocks assets helpers", func() {
	Describe("sanitizeSymbol", func() {
		DescribeTable("returns a Ledger-valid base or empty",
			func(in, want string) {
				Expect(sanitizeSymbol(in)).To(Equal(want))
			},
			Entry("uppercase passthrough", "USDT", "USDT"),
			Entry("lowercase uppercased", "xDAI", "XDAI"),
			Entry("digit-first prefix stripped", "1INCH", "INCH"),
			Entry("punctuation dropped", "BQ5R MY TOKEN", "BQ5RMYTOKEN"),
			Entry("hyphens & underscores dropped", "ETH-AETH_SEPOLIA", "ETHAETHSEPOLIA"),
			Entry("truncated at 17 chars", "ABCDEFGHIJKLMNOPQRSTUV", "ABCDEFGHIJKLMNOPQ"),
			Entry("only digits yields empty", "123", ""),
			Entry("empty input", "", ""),
		)
	})

	Describe("canonicalAsset", func() {
		It("appends precision when non-zero", func() {
			Expect(canonicalAsset("USDT", 6, false)).To(Equal("USDT/6"))
		})
		It("omits suffix when precision is zero", func() {
			Expect(canonicalAsset("JPY", 0, false)).To(Equal("JPY"))
		})
		It("returns empty when sanitisation fails", func() {
			Expect(canonicalAsset("...", 2, false)).To(BeEmpty())
		})
		It("appends _TEST for testnet assets", func() {
			Expect(canonicalAsset("ETH", 18, true)).To(Equal("ETH_TEST/18"))
		})
		It("preserves the full 17-char base when appending _TEST", func() {
			Expect(canonicalAsset("ABCDEFGHIJKLMNOPQ", 18, true)).To(Equal("ABCDEFGHIJKLMNOPQ_TEST/18"))
		})
	})

	Describe("buildAssetInfo", func() {
		It("uses onchain.decimals for fungible tokens", func() {
			info, ok := buildAssetInfo(client.Asset{
				LegacyID:      "USDT_ERC20",
				DisplaySymbol: "USDT",
				BlockchainID:  "chain-eth",
				AssetClass:    client.AssetClassFT,
				Onchain: &client.AssetOnchain{
					Symbol:    "USDT",
					Address:   "0xdAC17F958D2ee523a2206206994597C13D831ec7",
					Decimals:  6,
					Standards: []string{"ERC20"},
				},
			}, false)
			Expect(ok).To(BeTrue())
			Expect(info.Asset).To(Equal("USDT/6"))
			Expect(info.Precision).To(Equal(6))
			Expect(info.BlockchainID).To(Equal("chain-eth"))
			Expect(info.Metadata[MetadataPrefix+"legacy_id"]).To(Equal("USDT_ERC20"))
			Expect(info.Metadata[MetadataPrefix+"contract_address"]).To(Equal("0xdAC17F958D2ee523a2206206994597C13D831ec7"))
			Expect(info.Metadata[MetadataPrefix+"token_standard"]).To(Equal("ERC20"))
			Expect(info.Metadata[MetadataPrefix+"blockchain_id"]).To(Equal("chain-eth"))
		})

		It("uses top-level decimals for FIAT", func() {
			d := 2
			info, ok := buildAssetInfo(client.Asset{
				LegacyID:      "USD",
				DisplaySymbol: "USD",
				AssetClass:    client.AssetClassFiat,
				Decimals:      &d,
			}, false)
			Expect(ok).To(BeTrue())
			Expect(info.Asset).To(Equal("USD/2"))
		})

		It("falls back to onchain.symbol when displaySymbol is empty", func() {
			info, ok := buildAssetInfo(client.Asset{
				LegacyID:   "USDC_NEW",
				AssetClass: client.AssetClassFT,
				Onchain: &client.AssetOnchain{
					Symbol:   "USDC",
					Decimals: 6,
				},
			}, false)
			Expect(ok).To(BeTrue())
			Expect(info.Asset).To(Equal("USDC/6"))
		})

		It("sanitises a digit-first displaySymbol", func() {
			info, ok := buildAssetInfo(client.Asset{
				LegacyID:      "1INCH",
				DisplaySymbol: "1INCH",
				AssetClass:    client.AssetClassFT,
				Onchain:       &client.AssetOnchain{Decimals: 18},
			}, false)
			Expect(ok).To(BeTrue())
			Expect(info.Asset).To(Equal("INCH/18"))
		})

		It("emits verified+features metadata flags", func() {
			info, ok := buildAssetInfo(client.Asset{
				LegacyID:      "USDC",
				DisplaySymbol: "USDC",
				AssetClass:    client.AssetClassFT,
				Onchain:       &client.AssetOnchain{Decimals: 6},
				Metadata: &client.AssetSpecMetadata{
					Verified: true,
					Features: []string{"STABLECOIN"},
				},
			}, false)
			Expect(ok).To(BeTrue())
			Expect(info.Metadata[MetadataPrefix+"verified"]).To(Equal("true"))
			Expect(info.Metadata[MetadataPrefix+"features"]).To(Equal("STABLECOIN"))
			Expect(info.Metadata).ToNot(HaveKey(MetadataPrefix + "testnet"))
		})

		It("segregates testnet assets and stamps the testnet metadata flag", func() {
			info, ok := buildAssetInfo(client.Asset{
				LegacyID:      "ETH_TEST5",
				DisplaySymbol: "ETH",
				BlockchainID:  "chain-eth-sepolia",
				AssetClass:    client.AssetClassNative,
				Onchain:       &client.AssetOnchain{Decimals: 18},
			}, true)
			Expect(ok).To(BeTrue())
			Expect(info.Asset).To(Equal("ETH_TEST/18"))
			Expect(info.Metadata).To(HaveKeyWithValue(MetadataPrefix+"testnet", "true"))
		})

		It("skips deprecated assets", func() {
			_, ok := buildAssetInfo(client.Asset{
				LegacyID:      "OLD",
				DisplaySymbol: "OLD",
				AssetClass:    client.AssetClassFT,
				Onchain:       &client.AssetOnchain{Decimals: 18},
				Metadata:      &client.AssetSpecMetadata{Deprecated: true},
			}, false)
			Expect(ok).To(BeFalse())
		})

		DescribeTable("skips non-fungible / virtual classes",
			func(class string) {
				_, ok := buildAssetInfo(client.Asset{
					LegacyID:      "X",
					DisplaySymbol: "X",
					AssetClass:    class,
					Onchain:       &client.AssetOnchain{Decimals: 0},
				}, false)
				Expect(ok).To(BeFalse())
			},
			Entry("NFT", client.AssetClassNFT),
			Entry("SFT", client.AssetClassSFT),
			Entry("VIRTUAL", client.AssetClassVirtual),
		)

		It("skips when no decimals can be derived", func() {
			_, ok := buildAssetInfo(client.Asset{
				LegacyID:      "X",
				DisplaySymbol: "X",
				AssetClass:    client.AssetClassFT,
			}, false)
			Expect(ok).To(BeFalse())
		})

		It("skips when sanitisation yields empty symbol", func() {
			_, ok := buildAssetInfo(client.Asset{
				LegacyID:      "123",
				DisplaySymbol: "123",
				AssetClass:    client.AssetClassFT,
				Onchain:       &client.AssetOnchain{Decimals: 0},
			}, false)
			Expect(ok).To(BeFalse())
		})
	})
})
