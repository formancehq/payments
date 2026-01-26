package binance

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/binance/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Binance Plugin Suite")
}

var _ = Describe("Binance Plugin", func() {
	var (
		plg    *Plugin
		ctrl   *gomock.Controller
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
		}
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("install", func() {
		It("should report errors in config - apiKey", func(ctx SpecContext) {
			config := json.RawMessage(`{}`)
			_, err := New("binance", logger, config)
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("should report errors in config - secretKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test"}`)
			_, err := New("binance", logger, config)
			Expect(err.Error()).To(ContainSubstring("SecretKey"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			m := client.NewMockClient(ctrl)
			plg1 := &Plugin{
				Plugin: plugins.NewBasePlugin(),
				client: m,
				config: Config{
					APIKey:    "test-key",
					SecretKey: "test-secret",
				},
				logger: logger,
			}
			req := models.InstallRequest{}
			res, err := plg1.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("uninstall", func() {
		It("should return valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "test"}
			resp, err := plg.Uninstall(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})
	})

	Context("fetch next accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next balances", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next orders", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("create order", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreateOrderRequest{}
			_, err := plg.CreateOrder(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("cancel order", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CancelOrderRequest{}
			_, err := plg.CancelOrder(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("get order book", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.GetOrderBookRequest{}
			_, err := plg.GetOrderBook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("get quote", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.GetQuoteRequest{}
			_, err := plg.GetQuote(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("get tradable assets", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.GetTradableAssetsRequest{}
			_, err := plg.GetTradableAssets(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("get ticker", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.GetTickerRequest{}
			_, err := plg.GetTicker(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("get ohlc", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.GetOHLCRequest{}
			_, err := plg.GetOHLC(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("start order websocket", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.StartOrderWebSocketRequest{
				Config:  models.DefaultWebSocketConfig(),
				Handler: func(order models.PSPOrder) {},
			}
			_, err := plg.StartOrderWebSocket(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next payments", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("fetch next others", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create bank account", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create payout", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})
})
