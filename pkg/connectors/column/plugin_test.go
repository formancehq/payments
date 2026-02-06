package column

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Column Plugin Suite")
}

var _ = Describe("Column Plugin", func() {
	var (
		plg    connector.Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		connID = connector.ConnectorID{}
		ts     *httptest.Server
	)

	BeforeEach(func() {
		plg = &Plugin{
			Plugin: connector.NewBasePlugin(),
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
			req := connector.InstallRequest{}
			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("uninstall", func() {
		It("should return valid uninstall response", func(ctx SpecContext) {
			req := connector.UninstallRequest{ConnectorID: "test"}
			_, err := plg.Uninstall(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
	})

	Context("fetch next accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		// Other tests will be in accounts_test.go
	})

	Context("fetch next balances", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		// Other tests will be in balances_test.go
	})

	Context("fetch next external accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		// Other tests will be in external_accounts_test.go
	})

	Context("fetch next payments", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		// Other tests will be in payments_test.go
	})

	Context("fetch next others", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})

	})

	Context("create bank account", func() {
		It("should fail because not installed", func(ctx SpecContext) {
			req := connector.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
	})

	Context("create transfer", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		// Other tests will be in transfers_test.go
	})

	Context("reverse transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("poll transfer status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("create payout", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		// Other tests will be in payouts_test.go
	})

	Context("reverse payout", func() {
		It("should fail because not installed", func(ctx SpecContext) {
			req := connector.ReversePayoutRequest{}
			_, err := plg.ReversePayout(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
			// Other tests will be in reverse_payout_test.go
		})
	})

	Context("poll payout status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("create webhooks", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
	})

	Context("translate webhook", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
	})

	Context("When client is installed", func() {
		var plg connector.Plugin

		BeforeEach(func(ctx SpecContext) {
			config := json.RawMessage(fmt.Sprintf(`{"apiKey":"test","endpoint": "%s"}`, ts.URL))
			var err error
			p, err := New(connID, ProviderName, logger, config)
			Expect(err).To(BeNil())
			Expect(p.client).NotTo(BeNil())
			plg = p
		})

		It("should fail when fetching next others", func(ctx SpecContext) {
			req := connector.FetchNextOthersRequest{}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})

		It("should fail when reversing transfer", func(ctx SpecContext) {
			req := connector.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})

		It("should fail when polling transfer status", func(ctx SpecContext) {
			req := connector.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})

		It("should fail when polling transfer status", func(ctx SpecContext) {
			req := connector.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})

		It("should fail when polling payout status", func(ctx SpecContext) {
			req := connector.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})

	})

	It("should have the correct capabilities", func() {
		expectedCapabilities := []connector.Capability{
			connector.CAPABILITY_FETCH_ACCOUNTS,
			connector.CAPABILITY_FETCH_BALANCES,
			connector.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
			connector.CAPABILITY_FETCH_PAYMENTS,
			connector.CAPABILITY_CREATE_BANK_ACCOUNT,
			connector.CAPABILITY_CREATE_TRANSFER,
			connector.CAPABILITY_CREATE_PAYOUT,
			connector.CAPABILITY_CREATE_WEBHOOKS,
			connector.CAPABILITY_TRANSLATE_WEBHOOKS,
		}
		Expect(capabilities).To(HaveLen(len(expectedCapabilities)))
		Expect(capabilities).To(Equal(expectedCapabilities))
		// Verify each capability is present
		for _, expectedCap := range expectedCapabilities {
			Expect(capabilities).To(ContainElement(expectedCap))
		}
	})
})
