package kraken

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Kraken Plugin Balances", func() {
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

	Context("fetch next balances", func() {
		It("fetches balances successfully", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			sampleBalances := map[string]string{
				"XXBT": "1.5000000000",
				"XETH": "10.2500000000",
				"ZUSD": "5000.0000",
			}

			m.EXPECT().GetBalance(gomock.Any()).Return(sampleBalances, nil)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Balances).To(HaveLen(3))

			// Check that Kraken asset names are normalized
			assets := make(map[string]bool)
			for _, b := range res.Balances {
				assets[b.Asset] = true
				Expect(b.AccountReference).To(Equal("main"))
			}
			Expect(assets).To(HaveKey("BTC"))
			Expect(assets).To(HaveKey("ETH"))
			Expect(assets).To(HaveKey("USD"))
		})

		It("returns error on client failure", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetBalance(gomock.Any()).Return(nil, fmt.Errorf("client error"))

			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get balances"))
		})

		It("handles empty balances response", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetBalance(gomock.Any()).Return(map[string]string{}, nil)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Balances).To(BeEmpty())
		})

		It("skips balances with invalid amounts", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			invalidBalances := map[string]string{
				"XXBT": "1.5000000000",
				"XETH": "invalid-amount",
			}

			m.EXPECT().GetBalance(gomock.Any()).Return(invalidBalances, nil)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Balances).To(HaveLen(1))
			Expect(res.Balances[0].Asset).To(Equal("BTC"))
		})
	})

	Context("asset normalization", func() {
		It("normalizes Kraken asset names", func() {
			testCases := map[string]string{
				"XXBT":  "BTC",
				"XBT":   "BTC",
				"XETH":  "ETH",
				"ZUSD":  "USD",
				"ZEUR":  "EUR",
				"BTC":   "BTC",
				"ETH":   "ETH",
				"USDC":  "USDC",
			}

			for input, expected := range testCases {
				result := normalizeKrakenAsset(input)
				Expect(result).To(Equal(expected), "Expected %s to normalize to %s, got %s", input, expected, result)
			}
		})
	})
})
