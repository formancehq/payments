package gocardless_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless"
	"github.com/formancehq/payments/internal/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gocardless Plugin Suite")
}

var _ = Describe("Gocardless *Plugin", func() {
	var (
		plg    *gocardless.Plugin
		config json.RawMessage
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &gocardless.Plugin{}
		config = json.RawMessage(`{"accessToken":"abc123","endpoint":"example.com", "shouldFetchMandate":"true"}`)
	})

	Context("Install", func() {
		It("reports validation errors in the config", func(ctx SpecContext) {
			config := json.RawMessage(`{}`)
			_, err := gocardless.New("gocardless", logger, config)
			Expect(err.Error()).To(ContainSubstring("validation"))
		})

		It("reports validation errors in the config - accessToken", func(ctx SpecContext) {
			config := json.RawMessage(`{"endpoint":"example.com","shouldFetchMandate":"true"}`)
			_, err := gocardless.New("gocardless", logger, config)
			Expect(err.Error()).To(ContainSubstring("AccessToken"))
		})

		It("reports validation errors in the config - endpoint", func(ctx SpecContext) {
			config := json.RawMessage(`{"accessToken":"abc123","shouldFetchMandate":"true"}`)
			_, err := gocardless.New("gocardless", logger, config)
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})

		It("returns valid install response", func(ctx SpecContext) {
			_, err := gocardless.New("gocardless", logger, config)
			Expect(err).To(BeNil())

			res, err := plg.Install(context.Background(), models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow[0].Name).To(Equal("fetch_others"))
			Expect(res.Workflow).To(Equal(gocardless.Workflow()))
		})

		It("reports validation errors in the config - endpoint is not a string", func(ctx SpecContext) {
			config := json.RawMessage(`{"accessToken":"abc123","endpoint": 45,"shouldFetchMandate":"true"}`)
			_, err := gocardless.New("gocardless", logger, config)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal number into Go struct field Config.endpoint of type string"))
		})

		It("reports validation errors in the config - accessToken is not a string", func(ctx SpecContext) {
			config := json.RawMessage(`{"accessToken": 23,"endpoint": "example.com","shouldFetchMandate":"false"}`)
			_, err := gocardless.New("gocardless", logger, config)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal number into Go struct field Config.accessToken of type string"))
		})

	})

	Context("uninstall", func() {
		It("returns valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "dummyID"}
			_, err := plg.Uninstall(context.Background(), req)
			Expect(err).To(BeNil())
		})
	})

	Context("plugin name", func() {
		It("returns the plugin name", func(ctx SpecContext) {
			// Create a plugin instance with a known name
			plugin, err := gocardless.New("test-gocardless", logger, config)
			Expect(err).To(BeNil())
			Expect(plugin.Name()).To(Equal("test-gocardless"))
		})
	})

	Context("calling functions on uninstalled plugins", func() {
		It("fails when fetch next accounts is called before install", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextAccounts(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when fetch next balances is called before install", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextBalances(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when fetch next external accounts is called before install", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextExternalAccounts(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when fetch next payments is called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextPayments(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when fetch next others is called before install", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextOthers(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when create bank account is called before install", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when creating transfer is called before install", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when reverse transfer is called before install", func(ctx SpecContext) {
			req := models.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when poll transfer status is called before install", func(ctx SpecContext) {
			req := models.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when creating payout is called before install", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when reverse payout is called before install", func(ctx SpecContext) {
			req := models.ReversePayoutRequest{}
			_, err := plg.ReversePayout(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when poll payout status is called before install", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when create webhooks is called before install", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when translate webhook is called before install", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when verify webhook is called before install", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{}
			_, err := plg.VerifyWebhook(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("calling functions on unimplemented plugins", func() {
		var plg *gocardless.Plugin
		var err error

		BeforeEach(func() {
			plg, err = gocardless.New("gocardless", logger, config)
			Expect(err).To(BeNil())

			_, err := plg.Install(context.Background(), models.InstallRequest{})
			Expect(err).To(BeNil())
		})

		It("fails when FetchNextAccounts is called but not implemented", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextAccounts(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when FetchNextBalances is called but not implemented", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
			}

			_, err := plg.FetchNextBalances(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when CreateTransfer is called but not implemented", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when ReverseTransfer is called but not implemented", func(ctx SpecContext) {
			req := models.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when PollTransferStatus is called but not implemented", func(ctx SpecContext) {
			req := models.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when CreatePayout is called but not implemented", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when ReversePayout is called but not implemented", func(ctx SpecContext) {
			req := models.ReversePayoutRequest{}
			_, err := plg.ReversePayout(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when PollPayoutStatus is called but not implemented", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when CreateWebhooks is called but not implemented", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when TranslateWebhook is called but not implemented", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("fails when VerifyWebhook is called but not implemented", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{}
			_, err := plg.VerifyWebhook(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

	})

	Context("currencies", func() {
		It("should have the supported currencies", func() {
			expectedCurrencies := map[string]int{
				"AUD": currency.ISO4217Currencies["AUD"],
				"CAD": currency.ISO4217Currencies["CAD"],
				"DKK": currency.ISO4217Currencies["DKK"],
				"EUR": currency.ISO4217Currencies["EUR"],
				"GBP": currency.ISO4217Currencies["GBP"],
				"NZD": currency.ISO4217Currencies["NZD"],
				"SEK": currency.ISO4217Currencies["SEK"],
				"USD": currency.ISO4217Currencies["USD"],
			}

			Expect(gocardless.SupportedCurrenciesWithDecimal).To(Equal(expectedCurrencies))
			Expect(gocardless.SupportedCurrenciesWithDecimal).To(HaveLen(len(expectedCurrencies)))

		})
	})

	Context("capabilities", func() {
		It("should have the correct capabilities", func() {
			expectedCapabilities := []models.Capability{
				models.CAPABILITY_FETCH_OTHERS,
				models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
				models.CAPABILITY_FETCH_PAYMENTS,
				models.CAPABILITY_CREATE_BANK_ACCOUNT,
			}

			Expect(gocardless.Capabilities).To(Equal(expectedCapabilities))

			Expect(gocardless.Capabilities).To(HaveLen(len(expectedCapabilities)))

			// Verify each capability is present
			for _, expectedCap := range expectedCapabilities {
				Expect(gocardless.Capabilities).To(ContainElement(expectedCap))
			}
		})
	})

})
