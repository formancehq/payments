package tink

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Create User Link", func() {
	Context("create user link", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				client:   m,
				clientID: "test_client_id",
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should create user link successfully", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"
			redirectURL := "https://example.com/callback"
			callbackState := "test_state"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       callbackState,
			}

			expectedRequest := client.CreateTemporaryCodeRequest{
				UserID:   userID.String(),
				Username: "Test User",
				WantedScopes: []client.Scopes{
					client.SCOPES_AUTHORIZATION_READ,
					client.SCOPES_AUTHORIZATION_GRANT,
					client.SCOPES_CREDENTIALS_REFRESH,
					client.SCOPES_CREDENTIALS_READ,
					client.SCOPES_CREDENTIALS_WRITE,
					client.SCOPES_PROVIDERS_READ,
					client.SCOPES_USER_READ,
				},
			}

			m.EXPECT().CreateTemporaryAuthorizationCode(gomock.Any(), expectedRequest).Return(
				client.CreateTemporaryCodeResponse{
					Code: "temp_auth_code_123",
				},
				nil,
			)

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Link).To(ContainSubstring("https://link.tink.com/1.0/transactions"))
			Expect(resp.Link).To(ContainSubstring("client_id=test_client_id"))
			Expect(resp.Link).To(ContainSubstring("state=test_state"))
			Expect(resp.Link).To(ContainSubstring("authorization_code=temp_auth_code_123"))
			Expect(resp.Link).To(ContainSubstring("market=FR"))
			Expect(resp.Link).To(ContainSubstring("locale=fr_FR"))
			Expect(resp.Link).To(ContainSubstring("redirect_uri=https://example.com/callback"))
			Expect(resp.TemporaryLinkToken).ToNot(BeNil())
			Expect(resp.TemporaryLinkToken.Token).To(Equal("temp_auth_code_123"))
		})

		It("should return error when payment service user is nil", func(ctx SpecContext) {
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: nil,
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when formance redirect URL is nil", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: nil,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing formanceRedirectURL"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when formance redirect URL is empty", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"
			emptyURL := ""

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &emptyURL,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing formanceRedirectURL"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when callback state is empty", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"
			redirectURL := "https://example.com/callback"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       "",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing callBackState"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when payment service user address is nil", func(ctx SpecContext) {
			userID := uuid.New()
			locale := "fr_FR"
			redirectURL := "https://example.com/callback"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:      userID,
					Name:    "Test User",
					Address: nil,
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user country"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when payment service user address country is nil", func(ctx SpecContext) {
			userID := uuid.New()
			locale := "fr_FR"
			redirectURL := "https://example.com/callback"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: nil,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user country"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when payment service user country is not supported", func(ctx SpecContext) {
			userID := uuid.New()
			country := "XX" // Unsupported country
			locale := "fr_FR"
			redirectURL := "https://example.com/callback"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user country"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when payment service user contact details is nil", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			redirectURL := "https://example.com/callback"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: nil,
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user locale"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when payment service user locale is nil", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			redirectURL := "https://example.com/callback"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: nil,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user locale"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when payment service user locale is empty", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := ""
			redirectURL := "https://example.com/callback"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user locale"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when payment service user locale is not supported", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "xx_XX" // Unsupported locale
			redirectURL := "https://example.com/callback"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       "test_state",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user locale"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when client create temporary authorization code fails", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"
			locale := "fr_FR"
			redirectURL := "https://example.com/callback"
			callbackState := "test_state"

			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
					Address: &models.Address{
						Country: &country,
					},
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
				},
				FormanceRedirectURL: &redirectURL,
				CallBackState:       callbackState,
			}

			expectedRequest := client.CreateTemporaryCodeRequest{
				UserID:   userID.String(),
				Username: "Test User",
				WantedScopes: []client.Scopes{
					client.SCOPES_AUTHORIZATION_READ,
					client.SCOPES_AUTHORIZATION_GRANT,
					client.SCOPES_CREDENTIALS_REFRESH,
					client.SCOPES_CREDENTIALS_READ,
					client.SCOPES_CREDENTIALS_WRITE,
					client.SCOPES_PROVIDERS_READ,
					client.SCOPES_USER_READ,
				},
			}

			m.EXPECT().CreateTemporaryAuthorizationCode(gomock.Any(), expectedRequest).Return(
				client.CreateTemporaryCodeResponse{},
				errors.New("client error"),
			)

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})
	})
})
