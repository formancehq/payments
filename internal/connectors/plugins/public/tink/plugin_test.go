package tink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tink Plugin Suite")
}

var _ = Describe("Tink Plugin", func() {
	var (
		ctrl   *gomock.Controller
		m      *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("New", func() {
		It("should return error when config is invalid", func() {
			invalidConfig := json.RawMessage(`{"invalid": "config"}`)
			plg, err := New("test", logger, invalidConfig)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("ClientID"))
			Expect(plg).To(BeNil())
		})

		It("should return error when client_id is missing", func() {
			invalidConfig := json.RawMessage(`{
				"client_secret": "test-client-secret",
				"endpoint": "https://api.tink.com"
			}`)
			plg, err := New("test", logger, invalidConfig)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("ClientID"))
			Expect(plg).To(BeNil())
		})

		It("should return error when client_secret is missing", func() {
			invalidConfig := json.RawMessage(`{
				"client_id": "test-client-id",
				"endpoint": "https://api.tink.com"
			}`)
			plg, err := New("test", logger, invalidConfig)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("ClientSecret"))
			Expect(plg).To(BeNil())
		})

		It("should return error when endpoint is missing", func() {
			invalidConfig := json.RawMessage(`{
				"client_id": "test-client-id",
				"client_secret": "test-client-secret"
			}`)
			plg, err := New("test", logger, invalidConfig)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
			Expect(plg).To(BeNil())
		})

		It("should create plugin with valid config", func() {
			validConfig := json.RawMessage(`{
				"clientID": "test-client-id",
				"clientSecret": "test-client-secret",
				"endpoint": "https://api.tink.com"
			}`)
			plg, err := New("test", logger, validConfig)
			Expect(err).To(BeNil())
			Expect(plg).ToNot(BeNil())
			Expect(plg.Name()).To(Equal("test"))
		})
	})

	Context("Install", func() {
		BeforeEach(func() {
			plg = &Plugin{
				client: m,
			}
		})

		It("should return workflow", func(ctx SpecContext) {
			resp, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(resp.Workflow).ToNot(BeNil())
		})
	})

	Context("Uninstall", func() {
		BeforeEach(func() {
			plg = &Plugin{
				client: m,
			}
		})

		It("should return empty response", func(ctx SpecContext) {
			resp, err := plg.Uninstall(ctx, models.UninstallRequest{})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})
	})

	Context("CreateUserLink", func() {
		BeforeEach(func() {
			plg = &Plugin{
				client:   m,
				clientID: "test-client-id",
			}
		})

		It("should return error when plugin is not installed", func(ctx SpecContext) {
			plg.client = nil
			resp, err := plg.CreateUserLink(ctx, models.CreateUserLinkRequest{})
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when user is missing", func(ctx SpecContext) {
			resp, err := plg.CreateUserLink(ctx, models.CreateUserLinkRequest{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when user country is missing", func(ctx SpecContext) {
			userID := uuid.New()
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
				},
				RedirectURI: "https://example.com/callback",
			}
			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user country"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when user locale is missing", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
					Address: &models.Address{
						Country: &country,
					},
				},
				RedirectURI: "https://example.com/callback",
			}
			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user locale"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when redirect URI is missing", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
			}
			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing redirect URI"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when country is not supported", func(ctx SpecContext) {
			userID := uuid.New()
			country := "XX"
			locale := "fr_FR"
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				RedirectURI: "https://example.com/callback",
			}
			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user country"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when locale is not supported", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "xx_XX"
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				RedirectURI: "https://example.com/callback",
			}
			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user locale"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when CreateUser fails", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"
			m.EXPECT().CreateUser(gomock.Any(), userID.String(), "FR").Return(client.CreateUserResponse{}, fmt.Errorf("test error"))

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				RedirectURI: "https://example.com/callback",
			}
			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("test error"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when CreateTemporaryCode fails", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"
			createUserResp := client.CreateUserResponse{
				ExternalUserID: "test-external-user-id",
				UserID:         "test-user-id",
			}
			m.EXPECT().CreateUser(gomock.Any(), userID.String(), "FR").Return(createUserResp, nil)
			m.EXPECT().CreateTemporaryAuthorizationCode(gomock.Any(), client.CreateTemporaryCodeRequest{
				UserID:   "test-external-user-id",
				Username: "test-user",
				WantedScopes: []client.Scopes{
					client.SCOPES_AUTHORIZATION_READ,
					client.SCOPES_AUTHORIZATION_GRANT,
					client.SCOPES_CREDENTIALS_REFRESH,
					client.SCOPES_CREDENTIALS_READ,
					client.SCOPES_CREDENTIALS_WRITE,
					client.SCOPES_PROVIDERS_READ,
					client.SCOPES_USER_READ,
				},
			}).Return(client.CreateTemporaryCodeResponse{}, fmt.Errorf("temporary code error"))

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				RedirectURI: "https://example.com/callback",
			}
			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("temporary code error"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return valid user link response", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"
			createUserResp := client.CreateUserResponse{
				ExternalUserID: "test-external-user-id",
				UserID:         "test-user-id",
			}
			codeResp := client.CreateTemporaryCodeResponse{
				Code: "test-code",
			}

			m.EXPECT().CreateUser(gomock.Any(), userID.String(), "FR").Return(createUserResp, nil)
			m.EXPECT().CreateTemporaryAuthorizationCode(gomock.Any(), client.CreateTemporaryCodeRequest{
				UserID:   "test-external-user-id",
				Username: "test-user",
				WantedScopes: []client.Scopes{
					client.SCOPES_AUTHORIZATION_READ,
					client.SCOPES_AUTHORIZATION_GRANT,
					client.SCOPES_CREDENTIALS_REFRESH,
					client.SCOPES_CREDENTIALS_READ,
					client.SCOPES_CREDENTIALS_WRITE,
					client.SCOPES_PROVIDERS_READ,
					client.SCOPES_USER_READ,
				},
			}).Return(codeResp, nil)

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				RedirectURI: "https://example.com/callback",
			}
			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).To(BeNil())
			url, err := url.Parse(resp.Link)
			Expect(err).To(BeNil())
			query := url.Query()
			Expect(query.Get("client_id")).To(Equal("test-client-id"))
			Expect(query.Get("redirect_uri")).To(Equal("https://example.com/callback"))
			Expect(query.Get("authorization_code")).To(Equal("test-code"))
			Expect(query.Get("market")).To(Equal("FR"))
			Expect(query.Get("locale")).To(Equal("fr_FR"))
			Expect(query.Get("refreshable_items")).To(Equal("CHECKING_ACCOUNTS,CHECKING_TRANSACTIONS,SAVING_ACCOUNTS,SAVING_TRANSACTIONS,CREDITCARD_ACCOUNTS,CREDITCARD_TRANSACTIONS,TRANSFER_DESTINATIONS"))
			Expect(resp.TemporaryLinkToken).ToNot(BeNil())
			Expect(resp.TemporaryLinkToken.Token).To(Equal("test-code"))
		})
	})

	Context("CreateWebhooks", func() {
		BeforeEach(func() {
			plg = &Plugin{
				client:   m,
				clientID: "test-client-id",
			}
			plg.initWebhookConfig()
		})

		It("should return error when plugin is not installed", func(ctx SpecContext) {
			plg.client = nil
			resp, err := plg.CreateWebhooks(ctx, models.CreateWebhooksRequest{})
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return error when connector ID is missing", func(ctx SpecContext) {
			resp, err := plg.CreateWebhooks(ctx, models.CreateWebhooksRequest{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing connector ID"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return error when webhook base URL is missing", func(ctx SpecContext) {
			resp, err := plg.CreateWebhooks(ctx, models.CreateWebhooksRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing webhook base URL"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return error when CreateWebhook fails", func(ctx SpecContext) {
			m.EXPECT().CreateWebhook(gomock.Any(), gomock.Any(), "test-connector", gomock.Any()).Return(client.CreateWebhookResponse{}, fmt.Errorf("test error"))

			req := models.CreateWebhooksRequest{
				ConnectorID:    "test-connector",
				WebhookBaseUrl: "https://webhook.url",
			}
			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("test error"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return valid webhook configs", func(ctx SpecContext) {
			webhookResp := client.CreateWebhookResponse{
				ID:     "webhook-id",
				Secret: "webhook-secret",
			}
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountTransactionsModified, "test-connector", "https://webhook.url/account-transactions-modified").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountBookedTransactionsModified, "test-connector", "https://webhook.url/account-booked-transactions-modified").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountCreated, "test-connector", "https://webhook.url/account-created").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountUpdated, "test-connector", "https://webhook.url/account-updated").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.RefreshFinished, "test-connector", "https://webhook.url/refresh-finished").Return(webhookResp, nil)
			m.EXPECT().CreateWebhook(gomock.Any(), client.AccountTransactionsDeleted, "test-connector", "https://webhook.url/account-transactions-deleted").Return(webhookResp, nil)

			req := models.CreateWebhooksRequest{
				ConnectorID:    "test-connector",
				WebhookBaseUrl: "https://webhook.url",
			}
			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Configs).ToNot(BeEmpty())
			for _, cfg := range resp.Configs {
				Expect(cfg.Metadata).To(HaveKeyWithValue(webhookIDMetadataKey, "webhook-id"))
				Expect(cfg.Metadata).To(HaveKeyWithValue(webhookSecretMetadataKey, "webhook-secret"))
			}
		})
	})

	Context("VerifyWebhook", func() {
		BeforeEach(func() {
			plg = &Plugin{
				client: m,
			}
		})

		It("should return error when plugin is not installed", func(ctx SpecContext) {
			plg.client = nil
			resp, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{})
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when webhook is missing", func(ctx SpecContext) {
			resp, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing webhook"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when webhook config is missing", func(ctx SpecContext) {
			resp, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{},
				Config:  nil,
			})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing webhook config"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return error when signature header is missing", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
				},
				Config: &models.WebhookConfig{Metadata: map[string]string{}},
			}
			resp, err := plg.VerifyWebhook(ctx, req)
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
			resp, err := plg.VerifyWebhook(ctx, req)
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
			resp, err := plg.VerifyWebhook(ctx, req)
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
			resp, err := plg.VerifyWebhook(ctx, req)
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
			resp, err := plg.VerifyWebhook(ctx, req)
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
			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})
	})

	Context("TranslateWebhook", func() {
		BeforeEach(func() {
			plg = &Plugin{
				client: m,
			}
			plg.initWebhookConfig()
		})

		It("should return error when plugin is not installed", func(ctx SpecContext) {
			plg.client = nil
			resp, err := plg.TranslateWebhook(ctx, models.TranslateWebhookRequest{})
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return error for unsupported webhook event type", func(ctx SpecContext) {
			resp, err := plg.TranslateWebhook(ctx, models.TranslateWebhookRequest{
				Name: "unsupported",
			})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported webhook event type"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return error when webhook name is missing", func(ctx SpecContext) {
			resp, err := plg.TranslateWebhook(ctx, models.TranslateWebhookRequest{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing webhook name"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
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
