package coinbaseprime

import (
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Prime Plugin Tradable Assets", func() {
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
				PortfolioID: "test-portfolio-id",
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("GetTradableAssets", func() {
		var sampleProducts []client.Product

		BeforeEach(func() {
			sampleProducts = []client.Product{
				{
					ID:              "BTC-USD",
					BaseCurrency:   "BTC",
					QuoteCurrency:  "USD",
					BaseMinSize:    "0.0001",
					BaseMaxSize:    "1000",
					QuoteIncrement: "0.01",
					BaseIncrement:  "0.00000001",
					Status:         "online",
					TradingDisabled: false,
					CancelOnly:     false,
				},
				{
					ID:              "ETH-USD",
					BaseCurrency:   "ETH",
					QuoteCurrency:  "USD",
					BaseMinSize:    "0.001",
					BaseMaxSize:    "5000",
					QuoteIncrement: "0.01",
					BaseIncrement:  "0.00001",
					Status:         "online",
					TradingDisabled: false,
					CancelOnly:     false,
				},
				{
					ID:              "DOGE-USD",
					BaseCurrency:   "DOGE",
					QuoteCurrency:  "USD",
					BaseMinSize:    "1",
					BaseMaxSize:    "100000",
					QuoteIncrement: "0.0001",
					BaseIncrement:  "1",
					Status:         "offline",
					TradingDisabled: true,
					CancelOnly:     false,
				},
			}
		})

		It("fetches all tradable assets successfully", func(ctx SpecContext) {
			req := models.GetTradableAssetsRequest{}

			m.EXPECT().GetProducts(gomock.Any()).Return(sampleProducts, nil)

			resp, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(BeNil())
			// Should only return 2 assets (DOGE is disabled)
			Expect(resp.Assets).To(HaveLen(2))

			// Check first asset (BTC)
			Expect(resp.Assets[0].Pair).To(Equal("BTC/USD"))
			Expect(resp.Assets[0].BaseAsset).To(Equal("BTC"))
			Expect(resp.Assets[0].QuoteAsset).To(Equal("USD"))
			Expect(resp.Assets[0].MinOrderSize).To(Equal("0.0001"))
			Expect(resp.Assets[0].MaxOrderSize).To(Equal("1000"))
			Expect(resp.Assets[0].PricePrecision).To(Equal(2))
			Expect(resp.Assets[0].SizePrecision).To(Equal(8))
			Expect(resp.Assets[0].Status).To(Equal("online"))

			// Check second asset (ETH)
			Expect(resp.Assets[1].Pair).To(Equal("ETH/USD"))
			Expect(resp.Assets[1].BaseAsset).To(Equal("ETH"))
			Expect(resp.Assets[1].QuoteAsset).To(Equal("USD"))
			Expect(resp.Assets[1].PricePrecision).To(Equal(2))
			Expect(resp.Assets[1].SizePrecision).To(Equal(5))
		})

		It("filters by specific pairs", func(ctx SpecContext) {
			req := models.GetTradableAssetsRequest{
				Pairs: []string{"BTC/USD"},
			}

			m.EXPECT().GetProducts(gomock.Any()).Return(sampleProducts, nil)

			resp, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Assets).To(HaveLen(1))
			Expect(resp.Assets[0].Pair).To(Equal("BTC/USD"))
		})

		It("handles client error", func(ctx SpecContext) {
			req := models.GetTradableAssetsRequest{}

			m.EXPECT().GetProducts(gomock.Any()).Return(nil, fmt.Errorf("client error"))

			_, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get products"))
		})

		It("returns error when client is nil", func(ctx SpecContext) {
			plg.client = nil
			req := models.GetTradableAssetsRequest{}

			_, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})
})
