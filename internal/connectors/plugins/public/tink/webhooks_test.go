package tink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Webhooks", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			client: m,
		}
		plg.initWebhookConfig()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("create webhooks", func() {
		It("should return valid webhook configs", func(ctx SpecContext) {
			webhookResp := client.CreateWebhookResponse{
				ID:     "webhook-id",
				Secret: "webhook-secret",
			}
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountTransactionsModified, "connector-id", "https://webhook.url/account-transactions-modified").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountBookedTransactionsModified, "connector-id", "https://webhook.url/account-booked-transactions-modified").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountCreated, "connector-id", "https://webhook.url/account-created").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountUpdated, "connector-id", "https://webhook.url/account-updated").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.RefreshFinished, "connector-id", "https://webhook.url/refresh-finished").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountTransactionsDeleted, "connector-id", "https://webhook.url/account-transactions-deleted").Return(webhookResp, nil)

			req := models.CreateWebhooksRequest{
				ConnectorID:    "connector-id",
				WebhookBaseUrl: "https://webhook.url",
			}
			resp, err := plg.createWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Configs).ToNot(BeEmpty())
			for _, cfg := range resp.Configs {
				Expect(cfg.Metadata).To(HaveKeyWithValue(webhookIDMetadataKey, "webhook-id"))
				Expect(cfg.Metadata).To(HaveKeyWithValue(webhookSecretMetadataKey, "webhook-secret"))
			}
		})

		It("should return error if client returns error", func(ctx SpecContext) {
			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), "connector-id", gomock.Any()).Return(client.CreateWebhookResponse{}, fmt.Errorf("client error"))

			req := models.CreateWebhooksRequest{
				ConnectorID:    "connector-id",
				WebhookBaseUrl: "https://webhook.url",
			}
			resp, err := plg.createWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("client error"))
			Expect(resp.Configs).To(BeEmpty())
		})
	})

	Context("verify webhook", func() {
		It("should return error when signature header is missing", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
				},
				Config: &models.WebhookConfig{Metadata: map[string]string{}},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing tink signature header"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when signature header is invalid", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"X-Tink-Signature": {"invalid-signature"},
					},
				},
				Config: &models.WebhookConfig{Metadata: map[string]string{}},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid tink signature header"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when timestamp is invalid", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"X-Tink-Signature": {"t=invalid,v1=signature"},
					},
				},
				Config: &models.WebhookConfig{Metadata: map[string]string{}},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to parse timestamp"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when webhook is too old", func(ctx SpecContext) {
			oldTimestamp := time.Now().Add(-6 * time.Minute).Unix()
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"X-Tink-Signature": {fmt.Sprintf("t=%d,v1=signature", oldTimestamp)},
					},
				},
				Config: &models.WebhookConfig{Metadata: map[string]string{}},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("webhook created at"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when secret is missing", func(ctx SpecContext) {
			timestamp := time.Now().Unix()
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"X-Tink-Signature": {fmt.Sprintf("t=%d,v1=signature", timestamp)},
					},
					Body: []byte("test-body"),
				},
				Config: &models.WebhookConfig{
					Metadata: map[string]string{},
				},
			}
			resp, err := plg.verifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid signature"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should succeed when signature matches expected HMAC", func(ctx SpecContext) {
			secret := "test-secret"
			timestamp := time.Now().Unix()
			body := "test-body"
			messageToSign := fmt.Sprintf("%s.%s", fmt.Sprint(timestamp), body)
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(messageToSign))
			signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

			headerValue := fmt.Sprintf("t=%d,v1=%s", timestamp, signature)
			fmt.Println("DEBUG headerValue:", headerValue)

			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"X-Tink-Signature": {headerValue},
					},
					Body: []byte(body),
				},
				Config: &models.WebhookConfig{
					Metadata: map[string]string{
						webhookSecretMetadataKey: secret,
					},
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
				Name: string(client.AccountTransactionsModified),
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
