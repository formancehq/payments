package routable

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Routable Plugin Suite")
}

// validConfig is the canonical good config used across tests; specific specs
// alter it as needed before passing it to New.
var validConfig = json.RawMessage(`{"apiKey":"key","endpoint":"https://api.routable.com","actingTeamMember":"tm_1"}`)

var _ = Describe("Routable Plugin", func() {
	var (
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{Plugin: plugins.NewBasePlugin()}
	})

	Context("config validation", func() {
		It("rejects missing apiKey", func() {
			_, err := New("routable", logger, json.RawMessage(`{"actingTeamMember":"tm"}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})
		It("accepts a config without actingTeamMember (it's optional)", func() {
			p, err := New("routable", logger, json.RawMessage(`{"apiKey":"key"}`))
			Expect(err).To(BeNil())
			Expect(p.config.ActingTeamMember).To(BeEmpty())
		})
		It("falls back to the default endpoint when omitted", func() {
			p, err := New("routable", logger, json.RawMessage(`{"apiKey":"key"}`))
			Expect(err).To(BeNil())
			Expect(p.config.resolvedEndpoint()).To(Equal("https://api.routable.com"))
		})
	})

	Context("install", func() {
		It("returns the documented workflow", func() {
			p, err := New("routable", logger, validConfig)
			Expect(err).To(BeNil())
			res, err := p.Install(context.Background(), models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("uninstall", func() {
		It("returns an empty response", func() {
			_, err := plg.Uninstall(context.Background(), models.UninstallRequest{})
			Expect(err).To(BeNil())
		})
	})

	Context("calls before install", func() {
		It("returns ErrNotYetInstalled for FetchNextAccounts", func() {
			_, err := plg.FetchNextAccounts(context.Background(), models.FetchNextAccountsRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("returns ErrNotYetInstalled for FetchNextBalances", func() {
			_, err := plg.FetchNextBalances(context.Background(), models.FetchNextBalancesRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("returns ErrNotYetInstalled for FetchNextExternalAccounts", func() {
			_, err := plg.FetchNextExternalAccounts(context.Background(), models.FetchNextExternalAccountsRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("returns ErrNotYetInstalled for FetchNextPayments", func() {
			_, err := plg.FetchNextPayments(context.Background(), models.FetchNextPaymentsRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("returns ErrNotYetInstalled for CreateTransfer", func() {
			_, err := plg.CreateTransfer(context.Background(), models.CreateTransferRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("returns ErrNotYetInstalled for CreatePayout", func() {
			_, err := plg.CreatePayout(context.Background(), models.CreatePayoutRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("returns ErrNotYetInstalled for PollTransferStatus", func() {
			_, err := plg.PollTransferStatus(context.Background(), models.PollTransferStatusRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("returns ErrNotYetInstalled for PollPayoutStatus", func() {
			_, err := plg.PollPayoutStatus(context.Background(), models.PollPayoutStatusRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("base plugin inheritance", func() {
		It("returns ErrNotImplemented for unimplemented methods", func() {
			_, err := plg.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{})
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
			_, err = plg.CreateWebhooks(context.Background(), models.CreateWebhooksRequest{})
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
			_, err = plg.TranslateWebhook(context.Background(), models.TranslateWebhookRequest{})
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
			_, err = plg.ReversePayout(context.Background(), models.ReversePayoutRequest{})
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})
})
