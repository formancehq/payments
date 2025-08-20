package column

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Column Plugin Suite")
}

var _ = Describe("Column Plugin", func() {
	var (
		plg    models.Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		connID = models.ConnectorID{}
		ts     *httptest.Server
	)

	BeforeEach(func() {
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
		}

		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"webhook_urls": []}`))
			Expect(err).To(BeNil())
		}))
	})

	AfterEach(func() {
		ts.Close()
	})

	Context("install", func() {
		It("reports validation errors in the config", func(ctx SpecContext) {
			config := json.RawMessage(`{}`)
			_, err := New(connID, ProviderName, logger, config)
			Expect(err.Error()).To(ContainSubstring("validation"))
		})

		It("should report errors in config - apiKey", func(ctx SpecContext) {
			config := json.RawMessage(fmt.Sprintf(`{"endpoint": "%s"}`, ts.URL))
			_, err := New(connID, ProviderName, logger, config)
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("should report errors in config - endpoint", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test"}`)
			_, err := New(connID, ProviderName, logger, config)
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})

		It("should report errors in config - endpoint not url", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test", "endpoint": "fake"}`)
			_, err := New(connID, ProviderName, logger, config)
			fmt.Println(err.Error())
			Expect(err.Error()).To(ContainSubstring("Field validation"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			config := json.RawMessage(fmt.Sprintf(`{"apiKey": "test","endpoint": "%s"}`, ts.URL))
			plg, err := New(connID, ProviderName, logger, config)

			Expect(err).To(BeNil())
			req := models.InstallRequest{}
			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("uninstall", func() {
		It("should return valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "test"}
			_, err := plg.Uninstall(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		// Other tests will be in accounts_test.go
	})

	Context("fetch next balances", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		// Other tests will be in balances_test.go
	})

	Context("fetch next external accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		// Other tests will be in external_accounts_test.go
	})

	Context("fetch next payments", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		// Other tests will be in payments_test.go
	})

	Context("fetch next others", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

	})

	Context("create bank account", func() {
		It("should fail because not installed", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("create transfer", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		// Other tests will be in transfers_test.go
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
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		// Other tests will be in payouts_test.go
	})

	Context("reverse payout", func() {
		It("should fail because not installed", func(ctx SpecContext) {
			req := models.ReversePayoutRequest{}
			_, err := plg.ReversePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
			// Other tests will be in reverse_payout_test.go
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
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("translate webhook", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("When client is installed", func() {
		var plg models.Plugin

		BeforeEach(func(ctx SpecContext) {
			config := json.RawMessage(fmt.Sprintf(`{"apiKey":"test","endpoint": "%s"}`, ts.URL))
			var err error
			p, err := New(connID, ProviderName, logger, config)
			Expect(err).To(BeNil())
			Expect(p.client).NotTo(BeNil())
			plg = p
		})

		It("should fail when fetching next others", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("should fail when reversing transfer", func(ctx SpecContext) {
			req := models.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("should fail when polling transfer status", func(ctx SpecContext) {
			req := models.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("should fail when polling transfer status", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("should fail when polling payout status", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

	})

	It("should have the correct capabilities", func() {
		expectedCapabilities := []models.Capability{
			models.CAPABILITY_FETCH_ACCOUNTS,
			models.CAPABILITY_FETCH_BALANCES,
			models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
			models.CAPABILITY_FETCH_PAYMENTS,
			models.CAPABILITY_CREATE_BANK_ACCOUNT,
			models.CAPABILITY_CREATE_TRANSFER,
			models.CAPABILITY_CREATE_PAYOUT,
			models.CAPABILITY_CREATE_WEBHOOKS,
			models.CAPABILITY_TRANSLATE_WEBHOOKS,
		}
		Expect(capabilities).To(HaveLen(len(expectedCapabilities)))
		Expect(capabilities).To(Equal(expectedCapabilities))
		// Verify each capability is present
		for _, expectedCap := range expectedCapabilities {
			Expect(capabilities).To(ContainElement(expectedCap))
		}
	})
})
