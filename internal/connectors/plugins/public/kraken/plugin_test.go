package kraken

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kraken Plugin Suite")
}

var _ = Describe("Kraken Plugin", func() {
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
		It("should report errors in config - endpoint", func(ctx SpecContext) {
			config := json.RawMessage(`{}`)
			_, err := New("kraken", logger, config)
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})

		It("should report errors in config - publicKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"endpoint": "https://api.kraken.com"}`)
			_, err := New("kraken", logger, config)
			Expect(err.Error()).To(ContainSubstring("PublicKey"))
		})

		It("should report errors in config - privateKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"endpoint": "https://api.kraken.com", "publicKey": "test"}`)
			_, err := New("kraken", logger, config)
			Expect(err.Error()).To(ContainSubstring("PrivateKey"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			m := client.NewMockClient(ctrl)
			plg1 := &Plugin{
				Plugin: plugins.NewBasePlugin(),
				client: m,
				config: Config{
					Endpoint:   "https://api.kraken.com",
					PublicKey:  "test-public",
					PrivateKey: "test-private",
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
