package plaid

import (
	"errors"
	"time"

	"github.com/formancehq/payments/ce/plugins/plaid/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Plaid *Plugin Update User Link", func() {
	Context("update user link", func() {
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

		It("should return an error - missing application name", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: uuid.New(),
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing application name"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing connection", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing connection"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing access token", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection:      &models.OpenBankingConnection{},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing access token"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing payment service user", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing payment service user locale", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user locale"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing payment service user country", func(ctx SpecContext) {
			locale := "en-US"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user country"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - unsupported country", func(ctx SpecContext) {
			locale := "en-US"
			country := "XX"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user country"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing redirect URI", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing redirect URI"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing formance redirect URL", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				ClientRedirectURL: &redirectURL,
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing formance redirect URL"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing open banking connections", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			formanceRedirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				ClientRedirectURL:   &redirectURL,
				FormanceRedirectURL: &formanceRedirectURL,
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing open banking connections"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing open banking connections metadata", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			formanceRedirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				ClientRedirectURL:        &redirectURL,
				FormanceRedirectURL:      &formanceRedirectURL,
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing open banking connections metadata"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing user token", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			formanceRedirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				ClientRedirectURL:   &redirectURL,
				FormanceRedirectURL: &formanceRedirectURL,
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					Metadata: map[string]string{},
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing user token"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - invalid locale", func(ctx SpecContext) {
			locale := "invalid-locale"
			country := "US"
			redirectURL := "https://example.com/callback"
			formanceRedirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				ClientRedirectURL:   &redirectURL,
				FormanceRedirectURL: &formanceRedirectURL,
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					Metadata: map[string]string{
						UserTokenMetadataKey: "user-token-123",
					},
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid locale"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - unsupported locale", func(ctx SpecContext) {
			locale := "xx-XX"
			country := "US"
			redirectURL := "https://example.com/callback"
			formanceRedirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"
			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken: &models.Token{Token: "access-token-123"},
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				ClientRedirectURL:   &redirectURL,
				FormanceRedirectURL: &formanceRedirectURL,
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					Metadata: map[string]string{
						UserTokenMetadataKey: "user-token-123",
					},
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid locale"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should update user link successfully", func(ctx SpecContext) {
			userID := uuid.New()
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			formanceRedirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"
			webhookURL := "https://example.com/webhook"
			attemptID := uuid.New()

			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken:  &models.Token{Token: "access-token-123"},
					ConnectionID: "item-123",
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				ClientRedirectURL:   &redirectURL,
				FormanceRedirectURL: &formanceRedirectURL,
				WebhookBaseURL:      webhookURL,
				AttemptID:           attemptID.String(),
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					Metadata: map[string]string{
						UserTokenMetadataKey: "user-token-123",
					},
				},
			}

			expectedReq := client.UpdateLinkTokenRequest{
				AttemptID:           attemptID.String(),
				ApplicationName:     "Test",
				UserID:              userID.String(),
				UserToken:           "user-token-123",
				Language:            "en",
				CountryCode:         "US",
				RedirectURI:         "https://example.com/callback",
				AccessToken:         "access-token-123",
				ItemID:              "item-123",
				WebhookBaseURL:      "https://example.com/webhook",
				FormanceRedirectURL: formanceRedirectURL,
			}

			expectedResp := client.UpdateLinkTokenResponse{
				LinkToken:     "link-token-123",
				HostedLinkUrl: "https://plaid.com/link",
				Expiration:    time.Now().Add(time.Hour),
			}

			m.EXPECT().UpdateLinkToken(gomock.Any(), expectedReq).Return(expectedResp, nil)

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Link).To(Equal("https://plaid.com/link"))
			Expect(resp.TemporaryLinkToken.Token).To(Equal("link-token-123"))
			Expect(resp.TemporaryLinkToken.ExpiresAt).To(Equal(expectedResp.Expiration))
		})

		It("should return an error - client update link token error", func(ctx SpecContext) {
			userID := uuid.New()
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			formanceRedirectURL := "https://caller.example.com/open-banking/connections/attempt-1/callback"

			req := models.UpdateUserLinkRequest{
				ApplicationName: "Test",
				Connection: &models.OpenBankingConnection{
					AccessToken:  &models.Token{Token: "access-token-123"},
					ConnectionID: "item-123",
				},
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				ClientRedirectURL:   &redirectURL,
				FormanceRedirectURL: &formanceRedirectURL,
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					Metadata: map[string]string{
						UserTokenMetadataKey: "user-token-123",
					},
				},
			}

			m.EXPECT().UpdateLinkToken(gomock.Any(), gomock.Any()).Return(client.UpdateLinkTokenResponse{}, errors.New("client error"))

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})
	})
})
