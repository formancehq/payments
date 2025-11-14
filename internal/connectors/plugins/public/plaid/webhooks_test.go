package plaid

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Plaid *Plugin Webhooks", func() {
	Context("create webhooks", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			p := &Plugin{
				client: m,
			}

			p.supportedWebhooks = map[string]supportedWebhook{
				"all": {
					urlPath: "/all",
					fn:      p.handleAllWebhook,
				},
			}

			plg = p
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should create webhooks successfully", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Configs).To(HaveLen(1))
			Expect(resp.Configs[0].Name).To(Equal("all"))
			Expect(resp.Configs[0].URLPath).To(Equal("/all"))
		})
	})

	Context("verify webhook", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - missing Plaid-Verification header", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
				},
			}

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid token"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return an error - multiple Plaid-Verification headers", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Plaid-Verification": {"token1", "token2"},
					},
				},
			}

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid token"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})
	})

	Context("translate webhook", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			p := &Plugin{client: m}
			p.supportedWebhooks = map[string]supportedWebhook{
				"all": {
					urlPath: "/all",
					fn:      p.handleAllWebhook,
				},
			}

			plg = p
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - unsupported webhook name", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "unsupported",
				Webhook: models.PSPWebhook{
					Body: []byte(`{}`),
				},
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported webhook"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should translate webhook successfully", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "all",
				Webhook: models.PSPWebhook{
					Body: []byte(`{"webhook_type": "TRANSACTIONS", "webhook_code": "SYNC_UPDATES_AVAILABLE"}`),
				},
			}

			// Mock the BaseWebhookTranslation method
			baseWebhook := client.BaseWebhooks{
				WebhookType: "TRANSACTIONS",
				WebhookCode: "SYNC_UPDATES_AVAILABLE",
			}

			m.EXPECT().BaseWebhookTranslation(req.Webhook.Body).Return(baseWebhook, nil)

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).ToNot(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - base webhook translation error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "all",
				Webhook: models.PSPWebhook{
					Body: []byte(`invalid json`),
				},
			}

			m.EXPECT().BaseWebhookTranslation(req.Webhook.Body).Return(client.BaseWebhooks{}, errors.New("translation error"))

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("translation error"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})
	})
})
