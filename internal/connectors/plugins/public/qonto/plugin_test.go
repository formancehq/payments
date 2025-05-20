package qonto_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Qonto *Plugin Suite")
}

var _ = Describe("Qonto *Plugin", func() {
	var (
		plg    *qonto.Plugin
		config json.RawMessage
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &qonto.Plugin{
			Plugin: plugins.NewBasePlugin(),
		}
		config = json.RawMessage(`{"clientID":"1234","apiKey":"abc123","endpoint":"example.com"}`)
	})

	Context("install", func() {
		It("reports validation errors in the config", func(ctx SpecContext) {
			config := json.RawMessage(`{}`)
			_, err := qonto.New("qonto", logger, config)
			Expect(err.Error()).To(ContainSubstring("validation"))
		})
		It("returns valid install response", func(ctx SpecContext) {
			_, err := qonto.New("qonto", logger, config)
			Expect(err).To(BeNil())
			res, err := plg.Install(context.Background(), models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(res.Workflow[0].Name).To(Equal("fetch_accounts"))
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
		It("fails when fetch next payments is called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextPayments(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when fetch next external accounts is called before install", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextExternalAccounts(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when creating transfer as it's not installed yet", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when creating payout as it's unimplemented", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

})
