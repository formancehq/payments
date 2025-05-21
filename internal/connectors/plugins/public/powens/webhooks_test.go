package powens

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin Webhooks", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
		plg.initWebhookConfig()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("create webhooks", func() {
		It("should return valid webhook configs", func(ctx SpecContext) {
			m.EXPECT().CreateWebhookAuth(gomock.Any(), "connector-id").Return("secret-key", nil)
			req := models.CreateWebhooksRequest{ConnectorID: "connector-id"}
			resp, err := plg.createWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Configs).ToNot(BeEmpty())
			for _, cfg := range resp.Configs {
				Expect(cfg.Metadata).To(HaveKeyWithValue("secret", "secret-key"))
			}
		})

		It("should return error if client returns error", func(ctx SpecContext) {
			m.EXPECT().CreateWebhookAuth(gomock.Any(), "connector-id").Return("", fmt.Errorf("client error"))
			req := models.CreateWebhooksRequest{ConnectorID: "connector-id"}
			resp, err := plg.createWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("client error"))
			Expect(resp.Configs).To(BeEmpty())
		})
	})

	Context("verify webhook", func() {
		It("should return error when signature date header is missing", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
				},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing powens signature date header"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when signature header is missing", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{"BI-Signature-Date": {"2024-01-01T00:00:00Z"}},
				},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing powens signature header"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when signature is invalid base64", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"BI-Signature-Date": {"2024-01-01T00:00:00Z"},
						"BI-Signature":      {"not-base64!"},
					},
				},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid powens signature header"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when secret key is missing", func(ctx SpecContext) {
			sig := base64.StdEncoding.EncodeToString([]byte("somesig"))
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"BI-Signature-Date": {"2024-01-01T00:00:00Z"},
						"BI-Signature":      {sig},
					},
					Body: []byte("{}"),
				},
				Config: &models.WebhookConfig{
					Metadata: map[string]string{},
				},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing powens secret key"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when signature does not match expected HMAC", func(ctx SpecContext) {
			// Prepare a valid base64 signature, but with a wrong secret
			secret := "right-secret"
			date := "2024-01-01T00:00:00Z"
			body := "test-body"
			fullURL := "https://webhook.url/path"
			// Compute signature with wrong secret
			sig := base64.StdEncoding.EncodeToString([]byte("invalidsig"))
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"BI-Signature-Date": {date},
						"BI-Signature":      {sig},
					},
					Body: []byte(body),
				},
				Config: &models.WebhookConfig{
					FullURL:  fullURL,
					Metadata: map[string]string{"secret": secret},
				},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid powens signature"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should succeed when signature matches expected HMAC", func(ctx SpecContext) {
			secret := "my-secret"
			date := "2024-01-01T00:00:00Z"
			body := "test-body"
			fullURL := "https://webhook.url/path"
			messageToSign := "POST." + fullURL + "." + date + "." + body
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(messageToSign))
			sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"BI-Signature-Date": {date},
						"BI-Signature":      {sig},
					},
					Body: []byte(body),
				},
				Config: &models.WebhookConfig{
					FullURL:  fullURL,
					Metadata: map[string]string{"secret": secret},
				},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).To(BeNil())
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

		It("should be ok for a supported event type", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: string(client.WebhookEventTypeUserCreated),
			}
			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Responses).To(BeNil())
		})

		It("should return nil responses for all supported event types", func(ctx SpecContext) {
			for eventType := range plg.supportedWebhooks {
				req := models.TranslateWebhookRequest{
					Name: string(eventType),
				}
				resp, err := plg.TranslateWebhook(ctx, req)
				Expect(err).To(BeNil(), "eventType: %s", eventType)
				Expect(resp.Responses).To(BeNil(), "eventType: %s", eventType)
			}
		})
	})
})
