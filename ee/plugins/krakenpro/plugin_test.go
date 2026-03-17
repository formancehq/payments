package krakenpro

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Krakenpro Plugin Suite")
}

var _ = Describe("Krakenpro Plugin", func() {
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
			config := json.RawMessage(`{"privateKey": "dGVzdA=="}`)
			_, err := New("krakenpro", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("should report errors in config - privateKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test"}`)
			_, err := New("krakenpro", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PrivateKey"))
		})

		It("should report errors in config - invalid privateKey base64", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test", "privateKey": "not-valid-base64!!!"}`)
			_, err := New("krakenpro", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("decode private key"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()

			m := client.NewMockClient(ctrl)
			p := &Plugin{
				Plugin: plugins.NewBasePlugin(),
				client: m,
				logger: logger,
				config: Config{APIKey: "test-key"},
			}

			m.EXPECT().GetBalance(gomock.Any()).Return(
				&client.BalanceResponse{
					Error:  nil,
					Result: map[string]string{"ZUSD": "100.00"},
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
			// Fiat fallback should also be present
			Expect(p.currencies).To(HaveKey("USD"))
			Expect(p.currencies["USD"]).To(Equal(2))
			// Account ref should be derived
			Expect(p.accountRef).To(HavePrefix("kraken-"))
		})

		It("should return error when credential validation fails", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()

			m := client.NewMockClient(ctrl)
			p := &Plugin{
				Plugin: plugins.NewBasePlugin(),
				client: m,
				logger: logger,
			}

			m.EXPECT().GetBalance(gomock.Any()).Return(
				nil,
				fmt.Errorf("EAPI:Invalid key"),
			)

			req := models.InstallRequest{}
			_, err := p.Install(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validating credentials"))
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
