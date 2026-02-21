package coinbaseprime

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coinbase Plugin Suite")
}

var _ = Describe("Coinbase Plugin", func() {
	var (
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
		}
	})

	Context("install", func() {
		It("should report errors in config - apiKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiSecret": "dGVzdA==", "passphrase": "test", "portfolioId": "portfolio-123"}`)
			_, err := New("coinbaseprime", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("should report errors in config - apiSecret", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test", "passphrase": "test", "portfolioId": "portfolio-123"}`)
			_, err := New("coinbaseprime", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APISecret"))
		})

		It("should report errors in config - passphrase", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test", "apiSecret": "dGVzdA==", "portfolioId": "portfolio-123"}`)
			_, err := New("coinbaseprime", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Passphrase"))
		})

		It("should report errors in config - portfolioId", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test", "apiSecret": "dGVzdA==", "passphrase": "test"}`)
			_, err := New("coinbaseprime", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PortfolioID"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()

			m := client.NewMockClient(ctrl)
			p := &Plugin{
				Plugin: plugins.NewBasePlugin(),
				client: m,
				logger: logger,
			}

			m.EXPECT().GetPortfolio(gomock.Any()).Return(
				&client.PortfolioResponse{
					Portfolio: client.Portfolio{
						ID:       "portfolio-123",
						EntityID: "entity-456",
					},
				},
				nil,
			)

			m.EXPECT().GetAssets(gomock.Any(), "entity-456").Return(
				&client.AssetsResponse{
					Assets: []client.Asset{
						{Symbol: "BTC", DecimalPrecision: "8"},
						{Symbol: "ETH", DecimalPrecision: "18"},
						{Symbol: "USDC", DecimalPrecision: "6"},
					},
				},
				nil,
			)

			req := models.InstallRequest{}
			res, err := p.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))

			// Verify currencies were loaded
			Expect(p.currencies).To(HaveKey("BTC"))
			Expect(p.currencies["BTC"]).To(Equal(8))
			Expect(p.currencies).To(HaveKey("ETH"))
			Expect(p.currencies["ETH"]).To(Equal(18))
			Expect(p.currencies).To(HaveKey("USDC"))
			Expect(p.currencies["USDC"]).To(Equal(6))
			// Fiat fallback should also be present
			Expect(p.currencies).To(HaveKey("USD"))
			Expect(p.currencies["USD"]).To(Equal(2))
		})

		It("should return error when portfolio fetch fails", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()

			m := client.NewMockClient(ctrl)
			p := &Plugin{
				Plugin: plugins.NewBasePlugin(),
				client: m,
				logger: logger,
			}

			m.EXPECT().GetPortfolio(gomock.Any()).Return(
				nil,
				json.Unmarshal([]byte("invalid"), nil),
			)

			req := models.InstallRequest{}
			_, err := p.Install(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("loading currencies"))
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

	Context("fetch next payments", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next others", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("fetch next external accounts", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
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

	Context("reverse transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("poll transfer status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
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

	Context("reverse payout", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.ReversePayoutRequest{}
			_, err := plg.ReversePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("poll payout status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create webhooks", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("translate webhook", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})
})
