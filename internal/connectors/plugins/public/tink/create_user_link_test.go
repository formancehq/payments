package tink

import (
	"fmt"
	"net/url"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin CreateUserLink", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			client:   m,
			clientID: "test-client-id",
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return valid user link response", func(ctx SpecContext) {
		userID := uuid.New()
		userResp := client.CreateUserResponse{
			ExternalUserID: userID.String(),
			UserID:         "test-user-id",
		}
		codeResp := client.CreateTemporaryCodeResponse{
			Code: "test-code",
		}

		m.EXPECT().CreateUser(gomock.Any(), userID.String(), "FR").Return(userResp, nil)

		m.EXPECT().CreateTemporaryAuthorizationCode(gomock.Any(), client.CreateTemporaryCodeRequest{
			UserID:   userID.String(),
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

		country := "FR"
		locale := "fr_FR"
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PSPPaymentServiceUser{
				ID:        userID,
				Name:      "test-user",
				CreatedAt: time.Now().UTC(),
				Address: &models.Address{
					Country: &country,
				},
				ContactDetails: &models.ContactDetails{
					Locale: &locale,
				},
			},
			RedirectURI: "https://example.com/callback",
		}

		resp, err := plg.createUserLink(ctx, req)
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

	It("should return error when country is missing", func(ctx SpecContext) {
		locale := "fr_FR"
		userID := uuid.New()
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PSPPaymentServiceUser{
				ID:        userID,
				Name:      "test-user",
				CreatedAt: time.Now().UTC(),
				ContactDetails: &models.ContactDetails{
					Locale: &locale,
				},
			},
			RedirectURI: "https://example.com/callback",
		}

		resp, err := plg.createUserLink(ctx, req)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("missing payment service user country"))
		Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
	})

	It("should return error when country is not supported", func(ctx SpecContext) {
		country := "XX"
		locale := "fr_FR"
		userID := uuid.New()
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PSPPaymentServiceUser{
				ID:        userID,
				Name:      "test-user",
				CreatedAt: time.Now().UTC(),
				Address: &models.Address{
					Country: &country,
				},
				ContactDetails: &models.ContactDetails{
					Locale: &locale,
				},
			},
			RedirectURI: "https://example.com/callback",
		}

		resp, err := plg.createUserLink(ctx, req)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("unsupported payment service user country"))
		Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
	})

	It("should return error when locale is missing", func(ctx SpecContext) {
		country := "FR"
		userID := uuid.New()
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PSPPaymentServiceUser{
				ID:        userID,
				Name:      "test-user",
				CreatedAt: time.Now().UTC(),
				Address: &models.Address{
					Country: &country,
				},
			},
			RedirectURI: "https://example.com/callback",
		}

		resp, err := plg.createUserLink(ctx, req)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("missing payment service user locale"))
		Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
	})

	It("should return error when locale is not supported", func(ctx SpecContext) {
		country := "FR"
		locale := "xx_XX"
		userID := uuid.New()
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PSPPaymentServiceUser{
				ID:        userID,
				Name:      "test-user",
				CreatedAt: time.Now().UTC(),
				Address: &models.Address{
					Country: &country,
				},
				ContactDetails: &models.ContactDetails{
					Locale: &locale,
				},
			},
			RedirectURI: "https://example.com/callback",
		}

		resp, err := plg.createUserLink(ctx, req)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("unsupported payment service user locale"))
		Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
	})

	It("should return error when CreateUser fails", func(ctx SpecContext) {
		country := "FR"
		locale := "fr_FR"
		userID := uuid.New()
		m.EXPECT().CreateUser(gomock.Any(), userID.String(), "FR").Return(client.CreateUserResponse{}, fmt.Errorf("test error"))

		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PSPPaymentServiceUser{
				ID:        userID,
				Name:      "test-user",
				CreatedAt: time.Now().UTC(),
				Address: &models.Address{
					Country: &country,
				},
				ContactDetails: &models.ContactDetails{
					Locale: &locale,
				},
			},
			RedirectURI: "https://example.com/callback",
		}

		resp, err := plg.createUserLink(ctx, req)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("test error"))
		Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
	})

	It("should return error when CreateTemporaryCode fails", func(ctx SpecContext) {
		// Mock CreateUser response
		createUserResp := client.CreateUserResponse{
			ExternalUserID: "test-external-user-id",
			UserID:         "test-user-id",
		}
		userID := uuid.New()
		m.EXPECT().CreateUser(gomock.Any(), userID.String(), "FR").Return(createUserResp, nil)

		// Mock CreateTemporaryCode failure
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

		country := "FR"
		locale := "fr_FR"
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PSPPaymentServiceUser{
				ID:        userID,
				Name:      "test-user",
				CreatedAt: time.Now().UTC(),
				Address: &models.Address{
					Country: &country,
				},
				ContactDetails: &models.ContactDetails{
					Locale: &locale,
				},
			},
			RedirectURI: "https://example.com/callback",
		}

		resp, err := plg.createUserLink(ctx, req)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("temporary code error"))
		Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
	})
})
