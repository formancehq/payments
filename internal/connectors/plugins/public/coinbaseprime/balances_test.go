package coinbaseprime

import (
	"encoding/json"
	"fmt"

	"github.com/coinbase-samples/prime-sdk-go/balances"
	"github.com/coinbase-samples/prime-sdk-go/model"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Prime Plugin Balances", func() {
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

	Context("fetch next balances", func() {
		var sampleBalances []*model.Balance

		BeforeEach(func() {
			sampleBalances = []*model.Balance{
				{
					Symbol: "BTC",
					Amount: "1.5",
				},
				{
					Symbol: "ETH",
					Amount: "10.25",
				},
				{
					Symbol: "USD",
					Amount: "5000.00",
				},
			}
		})

		It("fetches balances successfully", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetPortfolioBalances(gomock.Any()).Return(
				&balances.ListPortfolioBalancesResponse{
					Balances: sampleBalances,
				},
				nil,
			)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Balances).To(HaveLen(3))
			Expect(res.Balances[0].Asset).To(Equal("BTC"))
			Expect(res.Balances[1].Asset).To(Equal("ETH"))
			Expect(res.Balances[2].Asset).To(Equal("USD"))
		})

		It("returns error on client failure", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetPortfolioBalances(gomock.Any()).Return(
				nil,
				fmt.Errorf("client error"),
			)

			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get portfolio balances"))
		})

		It("handles empty balances response", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetPortfolioBalances(gomock.Any()).Return(
				&balances.ListPortfolioBalancesResponse{
					Balances: []*model.Balance{},
				},
				nil,
			)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Balances).To(BeEmpty())
		})

		It("skips balances with invalid amounts", func(ctx SpecContext) {
			invalidBalances := []*model.Balance{
				{
					Symbol: "BTC",
					Amount: "1.5",
				},
				{
					Symbol: "ETH",
					Amount: "invalid-amount",
				},
			}

			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetPortfolioBalances(gomock.Any()).Return(
				&balances.ListPortfolioBalancesResponse{
					Balances: invalidBalances,
				},
				nil,
			)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Balances).To(HaveLen(1))
			Expect(res.Balances[0].Asset).To(Equal("BTC"))
		})

		It("normalizes lowercase symbols to uppercase", func(ctx SpecContext) {
			lowercaseBalances := []*model.Balance{
				{
					Symbol: "btc",
					Amount: "1.5",
				},
				{
					Symbol: "eth",
					Amount: "10.25",
				},
			}

			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetPortfolioBalances(gomock.Any()).Return(
				&balances.ListPortfolioBalancesResponse{
					Balances: lowercaseBalances,
				},
				nil,
			)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Balances).To(HaveLen(2))
			Expect(res.Balances[0].Asset).To(Equal("BTC"))
			Expect(res.Balances[1].Asset).To(Equal("ETH"))
		})

		It("skips balances with empty symbols", func(ctx SpecContext) {
			emptySymbolBalances := []*model.Balance{
				{
					Symbol: "BTC",
					Amount: "1.5",
				},
				{
					Symbol: "",
					Amount: "10.25",
				},
			}

			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetPortfolioBalances(gomock.Any()).Return(
				&balances.ListPortfolioBalancesResponse{
					Balances: emptySymbolBalances,
				},
				nil,
			)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Balances).To(HaveLen(1))
			Expect(res.Balances[0].Asset).To(Equal("BTC"))
		})

		It("handles mixed case symbols", func(ctx SpecContext) {
			mixedCaseBalances := []*model.Balance{
				{
					Symbol: "Btc",
					Amount: "1.5",
				},
				{
					Symbol: "uSdC",
					Amount: "100.00",
				},
			}

			req := models.FetchNextBalancesRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetPortfolioBalances(gomock.Any()).Return(
				&balances.ListPortfolioBalancesResponse{
					Balances: mixedCaseBalances,
				},
				nil,
			)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Balances).To(HaveLen(2))
			Expect(res.Balances[0].Asset).To(Equal("BTC"))
			Expect(res.Balances[1].Asset).To(Equal("USDC"))
		})
	})
})
