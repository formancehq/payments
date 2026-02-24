package moneycorp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/moneycorp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Moneycorp *Plugin Suite")
}

var _ = Describe("Moneycorp *Plugin", func() {
	var (
		plg    *moneycorp.Plugin
		config json.RawMessage
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &moneycorp.Plugin{
			Plugin: connector.NewBasePlugin(),
		}
		config = json.RawMessage(`{"clientID":"1234","apiKey":"abc123","endpoint":"example.com"}`)
	})

	Context("install", func() {
		It("reports validation errors in the config", func(ctx SpecContext) {
			config := json.RawMessage(`{}`)
			_, err := moneycorp.New("moneycorp", logger, config)
			Expect(err.Error()).To(ContainSubstring("validation"))
		})
		It("returns valid install response", func(ctx SpecContext) {
			_, err := moneycorp.New("moneycorp", logger, config)
			Expect(err).To(BeNil())
			res, err := plg.Install(context.Background(), connector.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(res.Workflow[0].Name).To(Equal("fetch_accounts"))
		})
	})

	Context("uninstall", func() {
		It("returns valid uninstall response", func(ctx SpecContext) {
			req := connector.UninstallRequest{ConnectorID: "dummyID"}
			_, err := plg.Uninstall(context.Background(), req)
			Expect(err).To(BeNil())
		})
	})

	Context("calling functions on uninstalled plugins", func() {
		It("fails when fetch next accounts is called before install", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextAccounts(context.Background(), req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		It("fails when fetch next balances is called before install", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextBalances(context.Background(), req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		It("fails when fetch next external accounts is called before install", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextExternalAccounts(context.Background(), req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		It("fails when creating transfer is called before install", func(ctx SpecContext) {
			req := connector.CreateTransferRequest{}
			_, err := plg.CreateTransfer(context.Background(), req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
		It("fails when creating payout is called before install", func(ctx SpecContext) {
			req := connector.CreatePayoutRequest{}
			_, err := plg.CreatePayout(context.Background(), req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})
	})
})
