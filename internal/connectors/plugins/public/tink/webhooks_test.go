package tink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Tink *Plugin Webhooks", func() {
	Context("create webhooks", func() {
		var (
			ctrl *gomock.Controller
			plg  *Plugin
			m    *client.MockClient
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

		It("should create webhooks successfully", func(ctx SpecContext) {
			connectorID := "connector_123"
			webhookBaseURL := "https://example.com/webhook"

			req := models.CreateWebhooksRequest{
				ConnectorID:    connectorID,
				WebhookBaseUrl: webhookBaseURL,
			}

			// Expect calls for each supported webhook event type
			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_1",
					Secret: "secret_1",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_2",
					Secret: "secret_2",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_3",
					Secret: "secret_3",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_4",
					Secret: "secret_4",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_5",
					Secret: "secret_5",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_6",
					Secret: "secret_6",
				},
				nil,
			)

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Configs).To(HaveLen(6))
		})

		It("should return error when connector ID is empty", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID:    "",
				WebhookBaseUrl: "https://example.com/webhook",
			}

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing connector ID"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return error when webhook base URL is empty", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID:    "connector_123",
				WebhookBaseUrl: "",
			}

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing webhook base URL"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return error when client create webhook fails", func(ctx SpecContext) {
			connectorID := "connector_123"
			webhookBaseURL := "https://example.com/webhook"

			req := models.CreateWebhooksRequest{
				ConnectorID:    connectorID,
				WebhookBaseUrl: webhookBaseURL,
			}

			// Expect all webhook types to be created, with the last one failing
			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_1",
					Secret: "secret_1",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_2",
					Secret: "secret_2",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_3",
					Secret: "secret_3",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_4",
					Secret: "secret_4",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{
					ID:     "webhook_5",
					Secret: "secret_5",
				},
				nil,
			)

			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), connectorID, gomock.Any()).Return(
				client.CreateWebhookResponse{},
				errors.New("client error"),
			)

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to create webhook"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})
	})

	Context("verify webhook", func() {
		var (
			ctrl *gomock.Controller
			plg  *Plugin
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

		It("should verify webhook successfully", func(ctx SpecContext) {
			// Create a proper signature for the test
			body := []byte("test_body")
			secret := "test_secret"
			timestamp := fmt.Sprintf("%d", time.Now().Unix())

			// Create the message to sign
			messageToSign := fmt.Sprintf("%s.%s", timestamp, string(body))

			// Create HMAC signature
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(messageToSign))
			signature := hex.EncodeToString(mac.Sum(nil))

			// Format the header as expected by Tink
			header := fmt.Sprintf("t=%s, v1=%s", timestamp, signature)

			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"X-Tink-Signature": {header},
					},
					Body: body,
				},
				Config: &models.WebhookConfig{
					Metadata: map[string]string{
						"secret": secret,
					},
				},
			}

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})
	})

	Context("translate webhook", func() {
		var (
			ctrl *gomock.Controller
			plg  *Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				client: m,
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return error when webhook name is empty", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "",
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing webhook name"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return error when webhook event type is not supported", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "unsupported_event",
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported webhook event type"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})
	})
})
