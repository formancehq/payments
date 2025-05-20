package plaid

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/plaid/plaid-go/v34/plaid"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Plaid *Plugin Webhooks", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plgImpl := &Plugin{client: m}
		plgImpl.initWebhookConfig()
		plg = plgImpl
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("create webhooks", func() {
		It("should return valid webhook configs", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Configs).To(HaveLen(1))
			Expect(resp.Configs[0].Name).To(Equal("all"))
			Expect(resp.Configs[0].URLPath).To(Equal("/all"))
		})
	})

	Context("verify webhook", func() {
		It("should return error when plaid-verification header is missing", func(ctx SpecContext) {
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

		It("should return error when plaid-verification header has multiple values", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"plaid-verification": {"token1", "token2"},
					},
				},
			}

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid token"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when webhook verification key is not found", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"plaid-verification": {"eyJhbGciOiJFUzI1NiIsImtpZCI6ImtpZCIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
					},
				},
			}

			m.EXPECT().GetWebhookVerificationKey(gomock.Any(), "kid").Return(nil, errors.New("key not found"))

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("key not found"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})
	})

	Context("translate webhook", func() {
		It("should return error for unsupported webhook event type", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "unsupported",
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported webhook event type"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return error when base webhook translation fails", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "all",
				Webhook: models.PSPWebhook{
					Body: []byte("invalid json"),
				},
			}

			m.EXPECT().BaseWebhookTranslation(req.Webhook.Body).Return(client.BaseWebhooks{}, errors.New("invalid json"))

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid json"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should handle transactions webhook", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "all",
				Webhook: models.PSPWebhook{
					Body: []byte(`{
						"webhook_type": "TRANSACTIONS",
						"webhook_code": "SYNC_UPDATES_AVAILABLE",
						"item_id": "test-item-id"
					}`),
				},
			}

			baseWebhook := client.BaseWebhooks{
				WebhookType: plaid.WEBHOOKTYPE_TRANSACTIONS,
				WebhookCode: "SYNC_UPDATES_AVAILABLE",
				ItemID:      "test-item-id",
			}

			m.EXPECT().BaseWebhookTranslation(req.Webhook.Body).Return(baseWebhook, nil)

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Responses).To(BeEmpty())
		})
	})
})
