package kraken

import (
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Kraken Plugin Tradable Assets", func() {
	var (
		ctrl   *gomock.Controller
		m      *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			client: m,
			logger: logger,
			config: Config{
				Endpoint: "https://api.kraken.com",
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("GetTradableAssets", func() {
		var sampleAssetPairs map[string]client.AssetPair

		BeforeEach(func() {
			sampleAssetPairs = map[string]client.AssetPair{
				"XXBTZUSD": {
					Altname:      "XBTUSD",
					WSName:       "XBT/USD",
					Base:         "XXBT",
					Quote:        "ZUSD",
					PairDecimals: 1,
					LotDecimals:  8,
					OrderMin:     "0.0001",
					Status:       "online",
				},
				"XETHZUSD": {
					Altname:      "ETHUSD",
					WSName:       "ETH/USD",
					Base:         "XETH",
					Quote:        "ZUSD",
					PairDecimals: 2,
					LotDecimals:  8,
					OrderMin:     "0.001",
					Status:       "online",
				},
				"XDGZUSD": {
					Altname:      "DOGEUSD",
					WSName:       "DOGE/USD",
					Base:         "XXDG",
					Quote:        "ZUSD",
					PairDecimals: 5,
					LotDecimals:  8,
					OrderMin:     "50",
					Status:       "offline",
				},
			}
		})

		It("fetches all tradable assets successfully", func(ctx SpecContext) {
			req := models.GetTradableAssetsRequest{}

			m.EXPECT().GetAssetPairs(gomock.Any()).Return(sampleAssetPairs, nil)

			resp, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(BeNil())
			// Should only return 2 assets (DOGE is offline)
			Expect(resp.Assets).To(HaveLen(2))

			// Find BTC asset
			var btcAsset, ethAsset *models.TradableAsset
			for i := range resp.Assets {
				if resp.Assets[i].BaseAsset == "BTC" {
					btcAsset = &resp.Assets[i]
				}
				if resp.Assets[i].BaseAsset == "ETH" {
					ethAsset = &resp.Assets[i]
				}
			}

			Expect(btcAsset).ToNot(BeNil())
			Expect(btcAsset.Pair).To(Equal("XBT/USD"))
			Expect(btcAsset.BaseAsset).To(Equal("BTC"))
			Expect(btcAsset.QuoteAsset).To(Equal("USD"))
			Expect(btcAsset.MinOrderSize).To(Equal("0.0001"))
			Expect(btcAsset.PricePrecision).To(Equal(1))
			Expect(btcAsset.SizePrecision).To(Equal(8))
			Expect(btcAsset.Status).To(Equal("online"))

			Expect(ethAsset).ToNot(BeNil())
			Expect(ethAsset.Pair).To(Equal("ETH/USD"))
			Expect(ethAsset.BaseAsset).To(Equal("ETH"))
			Expect(ethAsset.QuoteAsset).To(Equal("USD"))
			Expect(ethAsset.PricePrecision).To(Equal(2))
		})

		It("filters by specific pairs using WSName format", func(ctx SpecContext) {
			// Supports filtering by WSName format (e.g., "XBT/USD")
			req := models.GetTradableAssetsRequest{
				Pairs: []string{"XBT/USD"},
			}

			m.EXPECT().GetAssetPairs(gomock.Any()).Return(sampleAssetPairs, nil)

			resp, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Assets).To(HaveLen(1))
			Expect(resp.Assets[0].BaseAsset).To(Equal("BTC"))
		})

		It("filters by specific pairs using altname format", func(ctx SpecContext) {
			// Supports filtering by altname format (e.g., "XBTUSD")
			req := models.GetTradableAssetsRequest{
				Pairs: []string{"XBTUSD"},
			}

			m.EXPECT().GetAssetPairs(gomock.Any()).Return(sampleAssetPairs, nil)

			resp, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Assets).To(HaveLen(1))
			Expect(resp.Assets[0].BaseAsset).To(Equal("BTC"))
		})

		It("filters by specific pairs using full key", func(ctx SpecContext) {
			// Supports filtering by the full Kraken pair key (e.g., "XXBTZUSD")
			req := models.GetTradableAssetsRequest{
				Pairs: []string{"XXBTZUSD"},
			}

			m.EXPECT().GetAssetPairs(gomock.Any()).Return(sampleAssetPairs, nil)

			resp, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Assets).To(HaveLen(1))
			Expect(resp.Assets[0].BaseAsset).To(Equal("BTC"))
		})

		It("handles client error", func(ctx SpecContext) {
			req := models.GetTradableAssetsRequest{}

			m.EXPECT().GetAssetPairs(gomock.Any()).Return(nil, fmt.Errorf("client error"))

			_, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get asset pairs"))
		})

		It("returns error when client is nil", func(ctx SpecContext) {
			plg.client = nil
			req := models.GetTradableAssetsRequest{}

			_, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("krakenToStandard", func() {
		It("converts Kraken asset names to standard names", func() {
			Expect(krakenToStandard("XXBT")).To(Equal("BTC"))
			Expect(krakenToStandard("XBT")).To(Equal("BTC"))
			Expect(krakenToStandard("ZUSD")).To(Equal("USD"))
			Expect(krakenToStandard("XETH")).To(Equal("ETH"))
			Expect(krakenToStandard("XXDG")).To(Equal("DOGE"))
			Expect(krakenToStandard("USDT")).To(Equal("USDT"))
		})

		It("handles unknown assets", func() {
			// Unknown 4-letter codes starting with X or Z get prefix stripped
			Expect(krakenToStandard("XABC")).To(Equal("ABC"))
			Expect(krakenToStandard("ZABC")).To(Equal("ABC"))
			// Others are returned as-is
			Expect(krakenToStandard("SOL")).To(Equal("SOL"))
			Expect(krakenToStandard("MATIC")).To(Equal("MATIC"))
		})
	})
})
