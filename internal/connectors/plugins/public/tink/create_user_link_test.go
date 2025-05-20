package tink

import (
	"fmt"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestCreateUserLink(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tink CreateUserLink Suite")
}

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

		m.EXPECT().CreateTemporaryCode(gomock.Any(), client.CreateTemporaryCodeRequest{
			UserID:   userID.String(),
			Username: "test-user",
		}).Return(codeResp, nil)

		country := "FR"
		locale := "fr_FR"
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PaymentServiceUser{
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
		Expect(resp.Link).To(Equal("https://link.tink.com/1.0/transactions/connect-accounts?client_id=test-client-id&redirect_uri=https://example.com/callback&authorization_code=test-code&market=FR&locale=fr_FR"))
		Expect(resp.TemporaryLinkToken).ToNot(BeNil())
		Expect(resp.TemporaryLinkToken.Token).To(Equal("test-code"))
	})

	It("should return error when country is missing", func(ctx SpecContext) {
		locale := "fr_FR"
		userID := uuid.New()
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PaymentServiceUser{
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
			PaymentServiceUser: &models.PaymentServiceUser{
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
			PaymentServiceUser: &models.PaymentServiceUser{
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
			PaymentServiceUser: &models.PaymentServiceUser{
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
			PaymentServiceUser: &models.PaymentServiceUser{
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
		m.EXPECT().CreateTemporaryCode(gomock.Any(), client.CreateTemporaryCodeRequest{
			UserID:   "test-external-user-id",
			Username: "test-user",
		}).Return(client.CreateTemporaryCodeResponse{}, fmt.Errorf("temporary code error"))

		country := "FR"
		locale := "fr_FR"
		req := models.CreateUserLinkRequest{
			PaymentServiceUser: &models.PaymentServiceUser{
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
