package gocardless_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
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
		config = json.RawMessage(`{"accessToken":"abc123","endpoint":"example.com"}`)
	})

	Context("Install", func() {
		It("reports validation errors in the config", func(ctx SpecContext) {
			config := json.RawMessage(`{}`)
			_, err := gocardless.New("gocardless", logger, config)
			Expect(err.Error()).To(ContainSubstring("validation"))
		})

		It("reports validation errors in the config - accessToken", func(ctx SpecContext) {
			config := json.RawMessage(`{"endpoint":"example.com"}`)
			_, err := gocardless.New("gocardless", logger, config)
			Expect(err.Error()).To(ContainSubstring("AccessToken"))
		})

		It("reports validation errors in the config - endpoint", func(ctx SpecContext) {
			config := json.RawMessage(`{"accessToken":"abc123"}`)
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
	})

	Context("uninstall", func() {
		It("returns valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "dummyID"}
			_, err := plg.Uninstall(context.Background(), req)
			Expect(err).To(BeNil())
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
	})

	Context("capabilities", func() {
		It("should have the correct capabilities", func() {
			expectedCapabilities := []models.Capability{
				models.CAPABILITY_FETCH_OTHERS,
				models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
				models.CAPABILITY_FETCH_PAYMENTS,
				models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION,
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
