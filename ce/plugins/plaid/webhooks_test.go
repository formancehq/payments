package plaid

import (
	"errors"

	"github.com/formancehq/payments/ce/plugins/plaid/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/plaid/plaid-go/v34/plaid"
	gomock "go.uber.org/mock/gomock"
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

	Context("session finished webhook", func() {
		var (
			ctrl *gomock.Controller
			p    *Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			p = &Plugin{client: m}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - missing attemptID", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Webhook: models.PSPWebhook{
					Body:        []byte(`{}`),
					QueryValues: map[string][]string{},
				},
			}

			m.EXPECT().TranslateSessionFinishedWebhook(req.Webhook.Body).Return(plaid.LinkSessionFinishedWebhook{}, nil)

			_, err := p.handleSessionFinishedWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing attemptID"))
		})

		It("should return an error - ambiguous formanceRedirectURL", func(ctx SpecContext) {
			attemptID := uuid.New()
			req := models.TranslateWebhookRequest{
				Webhook: models.PSPWebhook{
					Body: []byte(`{}`),
					QueryValues: map[string][]string{
						client.AttemptIDQueryParamID:           {attemptID.String()},
						client.FormanceRedirectURLQueryParamID: {"https://a.example.com", "https://b.example.com"},
					},
				},
			}

			m.EXPECT().TranslateSessionFinishedWebhook(req.Webhook.Body).Return(plaid.LinkSessionFinishedWebhook{}, nil)

			_, err := p.handleSessionFinishedWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid formanceRedirectURL"))
		})

		It("should fall back to the legacy redirect construction for a pre-FormanceRedirectURL Link session", func(ctx SpecContext) {
			// Simulates a Link session created by a version of this plugin
			// predating the FormanceRedirectURL query param: its registered
			// webhook URL carries only attemptID, and it can still finish
			// after this plugin is redeployed. That must not drop the
			// webhook - it should fall back to letting
			// FormanceOpenBankingRedirect reconstruct the legacy URL
			// (RedirectURL == "").
			attemptID := uuid.New()
			req := models.TranslateWebhookRequest{
				Webhook: models.PSPWebhook{
					Body: []byte(`{}`),
					QueryValues: map[string][]string{
						client.AttemptIDQueryParamID: {attemptID.String()},
					},
				},
			}

			webhook := plaid.LinkSessionFinishedWebhook{
				Status:       "SUCCESS",
				LinkToken:    "link-token-123",
				PublicTokens: &[]string{"public-token-123"},
			}

			m.EXPECT().TranslateSessionFinishedWebhook(req.Webhook.Body).Return(webhook, nil)
			m.EXPECT().FormanceOpenBankingRedirect(ctx, client.FormanceOpenBankingRedirectRequest{
				RedirectURL: "",
				LinkToken:   "link-token-123",
				PublicToken: "public-token-123",
				AttemptID:   attemptID,
			}).Return(nil)

			resp, err := p.handleSessionFinishedWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(HaveLen(1))
			Expect(resp[0].UserLinkSessionFinished).ToNot(BeNil())
			Expect(resp[0].UserLinkSessionFinished.AttemptID).To(Equal(attemptID))
			Expect(resp[0].UserLinkSessionFinished.Status).To(Equal(models.OpenBankingConnectionAttemptStatusCompleted))
		})

		It("should notify the caller-supplied formanceRedirectURL and report completed status", func(ctx SpecContext) {
			attemptID := uuid.New()
			redirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"
			req := models.TranslateWebhookRequest{
				Webhook: models.PSPWebhook{
					Body: []byte(`{}`),
					QueryValues: map[string][]string{
						client.AttemptIDQueryParamID:           {attemptID.String()},
						client.FormanceRedirectURLQueryParamID: {redirectURL},
					},
				},
			}

			webhook := plaid.LinkSessionFinishedWebhook{
				Status:       "SUCCESS",
				LinkToken:    "link-token-123",
				PublicTokens: &[]string{"public-token-123"},
			}

			m.EXPECT().TranslateSessionFinishedWebhook(req.Webhook.Body).Return(webhook, nil)
			m.EXPECT().FormanceOpenBankingRedirect(ctx, client.FormanceOpenBankingRedirectRequest{
				RedirectURL: redirectURL,
				LinkToken:   "link-token-123",
				PublicToken: "public-token-123",
				AttemptID:   attemptID,
			}).Return(nil)

			resp, err := p.handleSessionFinishedWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(HaveLen(1))
			Expect(resp[0].UserLinkSessionFinished).ToNot(BeNil())
			Expect(resp[0].UserLinkSessionFinished.AttemptID).To(Equal(attemptID))
			Expect(resp[0].UserLinkSessionFinished.Status).To(Equal(models.OpenBankingConnectionAttemptStatusCompleted))
		})

		It("should propagate an error from FormanceOpenBankingRedirect", func(ctx SpecContext) {
			attemptID := uuid.New()
			redirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"
			req := models.TranslateWebhookRequest{
				Webhook: models.PSPWebhook{
					Body: []byte(`{}`),
					QueryValues: map[string][]string{
						client.AttemptIDQueryParamID:           {attemptID.String()},
						client.FormanceRedirectURLQueryParamID: {redirectURL},
					},
				},
			}

			webhook := plaid.LinkSessionFinishedWebhook{
				Status:       "SUCCESS",
				LinkToken:    "link-token-123",
				PublicTokens: &[]string{"public-token-123"},
			}

			m.EXPECT().TranslateSessionFinishedWebhook(req.Webhook.Body).Return(webhook, nil)
			m.EXPECT().FormanceOpenBankingRedirect(ctx, gomock.Any()).Return(errors.New("redirect error"))

			_, err := p.handleSessionFinishedWebhook(ctx, req)
			Expect(err).To(MatchError("redirect error"))
		})
	})
})
