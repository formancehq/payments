package universal_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Universal *Plugin Suite")
}

var _ = Describe("Universal *Plugin", func() {
	var (
		ctrl   *gomock.Controller
		mc     *client.MockClient
		plg    *universal.Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		cfg    = json.RawMessage(`{"endpoint":"https://upstream.example","apiKey":"k"}`)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mc = client.NewMockClient(ctrl)
		mc.EXPECT().SetIdempotencyHeader(gomock.Any()).AnyTimes()
		var err error
		plg, err = universal.New("universal-test", logger, cfg)
		Expect(err).To(BeNil())
		universal.InjectClient(plg, mc)
	})

	AfterEach(func() { ctrl.Finish() })

	Context("config validation", func() {
		It("rejects empty payload", func() {
			_, err := universal.New("u", logger, json.RawMessage(`{}`))
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("validation"))
		})

		It("rejects bad endpoint URL", func() {
			_, err := universal.New("u", logger, json.RawMessage(`{"endpoint":"not-a-url","apiKey":"k"}`))
			Expect(err).NotTo(BeNil())
		})

		It("rejects unknown capability override", func() {
			_, err := universal.New("u", logger, json.RawMessage(`{"endpoint":"https://x","apiKey":"k","capabilityOverrides":"NONSENSE"}`))
			Expect(err).NotTo(BeNil())
		})
	})

	Context("install", func() {
		It("discovers capabilities and builds the workflow tree", func(ctx SpecContext) {
			mc.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{
				Supported: []string{"FETCH_ACCOUNTS", "FETCH_BALANCES", "FETCH_PAYMENTS"},
				Features:  client.Features{Pagination: "cursor"},
			}, nil)

			res, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
			names := taskNames(res.Workflow)
			Expect(names).To(ContainElements("fetch_accounts", "fetch_balances", "fetch_payments"))
			Expect(names).NotTo(ContainElement("fetch_orders"))
		})

		It("triggers BootstrapOnInstall when ORDERS or CONVERSIONS are declared", func(ctx SpecContext) {
			mc.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{
				Supported: []string{"FETCH_ACCOUNTS", "FETCH_ORDERS"},
			}, nil)
			_, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(plg.BootstrapOnInstall()).To(Equal([]models.TaskType{models.TASK_FETCH_ACCOUNTS}))
		})

		It("does not trigger BootstrapOnInstall otherwise", func(ctx SpecContext) {
			mc.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{
				Supported: []string{"FETCH_ACCOUNTS", "FETCH_PAYMENTS"},
			}, nil)
			_, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(plg.BootstrapOnInstall()).To(BeNil())
		})

		It("rejects unknown capability strings from the counterparty", func(ctx SpecContext) {
			mc.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{
				Supported: []string{"NOT_REAL"},
			}, nil)
			_, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).NotTo(BeNil())
		})

		It("requires WebhookSharedSecret when counterparty signs webhooks", func(ctx SpecContext) {
			mc.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{
				Supported: []string{"CREATE_WEBHOOKS", "TRANSLATE_WEBHOOKS"},
				Features:  client.Features{WebhookSignature: "hmac-sha256"},
			}, nil)
			_, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("webhookSharedSecret"))
		})

		It("propagates upstream errors verbatim", func(ctx SpecContext) {
			mc.EXPECT().GetCapabilities(gomock.Any()).Return(nil, errors.New("boom"))
			_, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("boom"))
		})

		It("rejects install when counterparty declares zero capabilities", func(ctx SpecContext) {
			mc.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{Supported: []string{}}, nil)
			_, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).NotTo(BeNil())
			Expect(errors.Is(err, models.ErrInvalidConfig)).To(BeTrue())
		})

		It("rejects install when overrides narrow to empty (operator typo)", func(ctx SpecContext) {
			plgOv, err := universal.New("u", logger, json.RawMessage(`{"endpoint":"https://x","apiKey":"k","capabilityOverrides":"FETCH_PAYMENTS"}`))
			Expect(err).To(BeNil())
			mcOv := client.NewMockClient(ctrl)
			mcOv.EXPECT().SetIdempotencyHeader(gomock.Any()).AnyTimes()
			universal.InjectClient(plgOv, mcOv)
			mcOv.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{
				Supported: []string{"FETCH_ACCOUNTS"},
			}, nil)
			_, err = plgOv.Install(ctx, models.InstallRequest{})
			Expect(err).NotTo(BeNil())
			Expect(errors.Is(err, models.ErrInvalidConfig)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("FETCH_PAYMENTS"))
		})

		It("honours valid overrides that intersect with declared", func(ctx SpecContext) {
			plgOv, err := universal.New("u", logger, json.RawMessage(`{"endpoint":"https://x","apiKey":"k","capabilityOverrides":"FETCH_ACCOUNTS"}`))
			Expect(err).To(BeNil())
			mcOv := client.NewMockClient(ctrl)
			mcOv.EXPECT().SetIdempotencyHeader(gomock.Any()).AnyTimes()
			universal.InjectClient(plgOv, mcOv)
			mcOv.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{
				Supported: []string{"FETCH_ACCOUNTS", "FETCH_PAYMENTS"},
			}, nil)
			res, err := plgOv.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
			names := taskNames(res.Workflow)
			Expect(names).To(ContainElement("fetch_accounts"))
			Expect(names).NotTo(ContainElement("fetch_payments"))
		})
	})

	Context("guards on uninstalled plugin", func() {
		It("FetchNextAccounts returns ErrNotYetInstalled", func(ctx SpecContext) {
			_, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("CreatePayout returns ErrNotYetInstalled", func(ctx SpecContext) {
			_, err := plg.CreatePayout(ctx, models.CreatePayoutRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("CreateWebhooks returns ErrNotYetInstalled", func(ctx SpecContext) {
			_, err := plg.CreateWebhooks(ctx, models.CreateWebhooksRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("guards after install — undeclared capability", func() {
		BeforeEach(func(ctx SpecContext) {
			mc.EXPECT().GetCapabilities(gomock.Any()).Return(&client.CapabilitiesResponse{
				Supported: []string{"FETCH_ACCOUNTS"},
			}, nil)
			_, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
		})

		It("FetchNextOrders returns ErrNotImplemented", func(ctx SpecContext) {
			_, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
		It("CreatePayout returns ErrNotImplemented", func(ctx SpecContext) {
			_, err := plg.CreatePayout(ctx, models.CreatePayoutRequest{})
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
		It("CreateBankAccount returns ErrNotImplemented", func(ctx SpecContext) {
			_, err := plg.CreateBankAccount(ctx, models.CreateBankAccountRequest{})
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})
})

func taskNames(tree models.ConnectorTasksTree) []string {
	var out []string
	var walk func([]models.ConnectorTaskTree)
	walk = func(nodes []models.ConnectorTaskTree) {
		for _, n := range nodes {
			out = append(out, n.Name)
			walk(n.NextTasks)
		}
	}
	walk(tree)
	return out
}
