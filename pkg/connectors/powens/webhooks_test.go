package powens

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/formancehq/payments/pkg/connectors/powens/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin Webhooks", func() {
	Context("create webhooks", func() {
		var (
			ctrl *gomock.Controller
			plg  connector.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				client: m,
				name:   "test-powens",
			}
			// Initialize webhook configuration
			plg.(*Plugin).initWebhookConfig()
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should create webhooks successfully", func(ctx SpecContext) {
			req := connector.CreateWebhooksRequest{
				WebhookBaseUrl: "https://webhook.example.com",
			}

			m.EXPECT().CreateWebhookAuth(gomock.Any(), "test-powens").Return("secret-key", nil)

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Configs).To(HaveLen(3))
		})

		It("should return an error - client create webhook auth error", func(ctx SpecContext) {
			req := connector.CreateWebhooksRequest{
				WebhookBaseUrl: "https://webhook.example.com",
			}

			m.EXPECT().CreateWebhookAuth(gomock.Any(), "test-powens").Return("", errors.New("client error"))

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(connector.CreateWebhooksResponse{}))
		})
	})

	Context("verify webhook", func() {
		var (
			plg connector.Plugin
		)

		BeforeEach(func() {
			plg = &Plugin{
				client: &client.MockClient{},
			}
		})

		It("should verify webhook successfully", func(ctx SpecContext) {
			// Create a valid signature
			secretKey := "secret-key"
			signatureDate := "2023-01-01T00:00:00Z"
			body := []byte("test-body")
			urlPath := "/user-created"

			messageToSign := "POST." + urlPath + "." + signatureDate + "." + string(body)
			mac := hmac.New(sha256.New, []byte(secretKey))
			mac.Write([]byte(messageToSign))
			expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

			req := connector.VerifyWebhookRequest{
				Webhook: connector.PSPWebhook{
					Headers: map[string][]string{
						"Bi-Signature-Date": {signatureDate},
						"Bi-Signature":      {expectedSignature},
					},
					Body: body,
				},
				Config: &connector.WebhookConfig{
					FullURL: "https://webhook.example.com" + urlPath,
					Metadata: map[string]string{
						webhookSecretMetadataKey: secretKey,
					},
				},
			}

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(connector.VerifyWebhookResponse{}))
		})

		It("should return an error - missing signature date header", func(ctx SpecContext) {
			req := connector.VerifyWebhookRequest{
				Webhook: connector.PSPWebhook{
					Headers: map[string][]string{},
					Body:    []byte("test-body"),
				},
			}

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing powens signature date header"))
			Expect(resp).To(Equal(connector.VerifyWebhookResponse{}))
		})
	})

	Context("translate webhook", func() {
		var (
			plg connector.Plugin
		)

		BeforeEach(func() {
			plg = &Plugin{
				client: &client.MockClient{},
			}
			// Initialize webhook configuration
			plg.(*Plugin).initWebhookConfig()
		})

		It("should return an error - unsupported webhook event type", func(ctx SpecContext) {
			req := connector.TranslateWebhookRequest{
				Name: "unsupported-event",
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported webhook event type"))
			Expect(resp).To(Equal(connector.TranslateWebhookResponse{}))
		})

		It("sets balance CreatedAt to account lastUpdate when present and converts to UTC from default Europe/Paris", func(ctx SpecContext) {
			body := []byte(`{"user":{"id":1},"connection":{"id":10,"state":"","accounts":[{"id":100,"id_user":1,"id_connection":10,"currency":{"id":"EUR","precision":2},"last_update":"2023-05-01 12:34:56","balance":"123.45","transactions":[]}]}}`)
			req := connector.TranslateWebhookRequest{
				Name:    string(client.WebhookEventTypeConnectionSynced),
				Webhook: connector.PSPWebhook{Body: body},
			}
			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			// Last responses include balance if present; find it
			var found bool
			for _, r := range resp.Responses {
				if r.Balance != nil {
					found = true
					Expect(r.Balance.CreatedAt.String()).To(Equal("2023-05-01 10:34:56 +0000 UTC"))
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Context("trim webhook", func() {
		var (
			plg connector.Plugin
		)

		BeforeEach(func() {
			plg = &Plugin{
				client: &client.MockClient{},
			}
			// Initialize webhook configuration
			plg.(*Plugin).initWebhookConfig()
		})

		It("should return an error - unsupported webhook event type", func(ctx SpecContext) {
			req := connector.TrimWebhookRequest{
				Config: &connector.WebhookConfig{
					Name: "unsupported-event",
				},
			}

			resp, err := plg.TrimWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported webhook event type"))
			Expect(resp).To(Equal(connector.TrimWebhookResponse{}))
		})

		It("should trim connection synced webhook successfully", func(ctx SpecContext) {
			req := connector.TrimWebhookRequest{
				Config: &connector.WebhookConfig{
					Name: string(client.WebhookEventTypeConnectionSynced),
				},
				Webhook: connector.PSPWebhook{
					Body: []byte(`{"user": {"id": 1}, "connection": {"id": 1, "state": "", "accounts": [{"id": 1, "transactions": [{"id": 1}, {"id": 2}]}]}}`),
				},
			}

			resp, err := plg.TrimWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Webhooks).To(HaveLen(1))
		})

		It("should trim connection synced webhook successfully even with empty transactions", func(ctx SpecContext) {
			req := connector.TrimWebhookRequest{
				Config: &connector.WebhookConfig{
					Name: string(client.WebhookEventTypeConnectionSynced),
				},
				Webhook: connector.PSPWebhook{
					Body: []byte(`{"user": {"id": 1}, "connection": {"id": 1, "state": "", "accounts": [{"id": 1}]}}`),
				},
			}

			resp, err := plg.TrimWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Webhooks).To(HaveLen(1))
		})

		It("should trim connection synced webhook successfully with last updated at as UTC", func(ctx SpecContext) {
			req := connector.TrimWebhookRequest{
				Config: &connector.WebhookConfig{
					Name: string(client.WebhookEventTypeConnectionSynced),
				},
				Webhook: connector.PSPWebhook{
					Body: []byte(`{"user": {"id": 1}, "connection": {"id": 1, "state": "", "last_update": "2021-10-20 19:00:00", "accounts": [{"id": 1}]}}`),
				},
			}

			resp, err := plg.TrimWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Webhooks).To(HaveLen(1))

			var webhook client.ConnectionSyncedWebhook
			err = json.Unmarshal(resp.Webhooks[0].Body, &webhook)
			Expect(err).To(BeNil())

			Expect(webhook.Connection.LastUpdate.String()).To(Equal("2021-10-20 17:00:00 +0000 UTC"))
		})

		It("should keep account balance after trimming", func(ctx SpecContext) {
			req := connector.TrimWebhookRequest{
				Config: &connector.WebhookConfig{
					Name: string(client.WebhookEventTypeConnectionSynced),
				},
				Webhook: connector.PSPWebhook{
					Body: []byte(`{"user":{"id":1},"connection":{"id":10,"state":"","accounts":[{"id":100,"id_user":1,"id_connection":10,"currency":{"id":"EUR","precision":2},"balance":"123.45","transactions":[]}]}}`),
				},
			}

			resp, err := plg.TrimWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Webhooks).To(HaveLen(1))

			var trimmed client.ConnectionSyncedWebhook
			err = json.Unmarshal(resp.Webhooks[0].Body, &trimmed)
			Expect(err).To(BeNil())
			Expect(trimmed.Connection.Accounts).To(HaveLen(1))
			Expect(trimmed.Connection.Accounts[0].Balance.String()).To(Equal("123.45"))
		})
	})
})
