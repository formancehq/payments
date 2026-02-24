package plaid

import (
	"errors"
	"time"

	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Plaid *Plugin Create User Link", func() {
	Context("create user link", func() {
		var (
			ctrl *gomock.Controller
			plg  connector.Plugin
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
			req := connector.CreateUserLinkRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID: uuid.New(),
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing application name"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - missing payment service user", func(ctx SpecContext) {
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - missing payment service user name", func(ctx SpecContext) {
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID: uuid.New(),
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user name"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - missing payment service user locale", func(ctx SpecContext) {
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user locale"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - missing payment service user country", func(ctx SpecContext) {
			locale := "en-US"
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user country"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - unsupported country", func(ctx SpecContext) {
			locale := "en-US"
			country := "XX"
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user country"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - missing redirect URI", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing redirect URI"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - missing open banking connections", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
				ClientRedirectURL: &redirectURL,
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing open banking connections"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - missing open banking connections metadata", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
				ClientRedirectURL:        &redirectURL,
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing open banking connections metadata"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - missing user token", func(ctx SpecContext) {
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
				ClientRedirectURL: &redirectURL,
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					Metadata: map[string]string{},
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing user token"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - invalid locale", func(ctx SpecContext) {
			locale := "invalid-locale"
			country := "US"
			redirectURL := "https://example.com/callback"
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
				ClientRedirectURL: &redirectURL,
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					Metadata: map[string]string{
						UserTokenMetadataKey: "user-token-123",
					},
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid locale"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should return an error - unsupported locale", func(ctx SpecContext) {
			locale := "xx-XX"
			country := "US"
			redirectURL := "https://example.com/callback"
			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   uuid.New(),
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
				ClientRedirectURL: &redirectURL,
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					Metadata: map[string]string{
						UserTokenMetadataKey: "user-token-123",
					},
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid locale"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})

		It("should create user link successfully", func(ctx SpecContext) {
			userID := uuid.New()
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"
			webhookURL := "https://example.com/webhook"
			attemptID := uuid.New()

			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   userID,
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
				ClientRedirectURL: &redirectURL,
				WebhookBaseURL:    webhookURL,
				AttemptID:         attemptID.String(),
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					Metadata: map[string]string{
						UserTokenMetadataKey: "user-token-123",
					},
				},
			}

			expectedReq := client.CreateLinkTokenRequest{
				ApplicationName: "Test",
				UserID:          userID.String(),
				UserToken:       "user-token-123",
				Language:        "en",
				CountryCode:     "US",
				RedirectURI:     "https://example.com/callback",
				WebhookBaseURL:  "https://example.com/webhook",
				AttemptID:       attemptID.String(),
			}

			expectedResp := client.CreateLinkTokenResponse{
				LinkToken:     "link-token-123",
				HostedLinkUrl: "https://plaid.com/link",
				Expiration:    time.Now().Add(time.Hour),
			}

			m.EXPECT().CreateLinkToken(gomock.Any(), expectedReq).Return(expectedResp, nil)

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Link).To(Equal("https://plaid.com/link"))
			Expect(resp.TemporaryLinkToken.Token).To(Equal("link-token-123"))
			Expect(resp.TemporaryLinkToken.ExpiresAt).To(Equal(expectedResp.Expiration))
		})

		It("should return an error - client create link token error", func(ctx SpecContext) {
			userID := uuid.New()
			locale := "en-US"
			country := "US"
			redirectURL := "https://example.com/callback"

			req := connector.CreateUserLinkRequest{
				ApplicationName: "Test",
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:   userID,
					Name: "John Doe",
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
					Address: &connector.Address{
						Country: &country,
					},
				},
				ClientRedirectURL: &redirectURL,
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					Metadata: map[string]string{
						UserTokenMetadataKey: "user-token-123",
					},
				},
			}

			m.EXPECT().CreateLinkToken(gomock.Any(), gomock.Any()).Return(client.CreateLinkTokenResponse{}, errors.New("client error"))

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(connector.CreateUserLinkResponse{}))
		})
	})
})
