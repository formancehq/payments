package plaid

import (
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Plaid *Plugin Create User Link", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("create user link", func() {
		var (
			sampleRequest models.CreateUserLinkRequest
			now           time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()
			locale := "en"
			country := "US"

			sampleRequest = models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
					Name: "John Doe",
					ContactDetails: &models.ContactDetails{
						Locale: &locale,
					},
					Address: &models.Address{
						Country: &country,
					},
				},
				RedirectURI:    "https://example.com/redirect",
				WebhookBaseURL: "https://example.com/webhook",
			}
		})

		It("should return error - validation error - missing payment service user", func(ctx SpecContext) {
			req := models.CreateUserLinkRequest{}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error - validation error - missing payment service user name", func(ctx SpecContext) {
			req := sampleRequest
			req.PaymentServiceUser.Name = ""

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user name"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error - validation error - missing payment service user locale", func(ctx SpecContext) {
			req := sampleRequest
			req.PaymentServiceUser.ContactDetails.Locale = nil

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user locale"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error - validation error - missing payment service user country", func(ctx SpecContext) {
			req := sampleRequest
			req.PaymentServiceUser.Address.Country = nil

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing payment service user country"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error - validation error - unsupported payment service user country", func(ctx SpecContext) {
			req := sampleRequest
			unsupportedCountry := "XX"
			req.PaymentServiceUser.Address.Country = &unsupportedCountry

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user country"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error - validation error - missing redirect URI", func(ctx SpecContext) {
			req := sampleRequest
			req.RedirectURI = ""

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing redirect URI"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error - create link token error", func(ctx SpecContext) {
			req := sampleRequest

			m.EXPECT().CreateLinkToken(gomock.Any(), client.CreateLinkTokenRequest{
				UserName:       req.PaymentServiceUser.Name,
				UserID:         req.PaymentServiceUser.ID.String(),
				Language:       "en",
				CountryCode:    *req.PaymentServiceUser.Address.Country,
				RedirectURI:    req.RedirectURI,
				WebhookBaseURL: req.WebhookBaseURL,
			}).Return(client.CreateLinkTokenResponse{}, errors.New("test error"))

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("test error"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := sampleRequest

			linkTokenResp := client.CreateLinkTokenResponse{
				LinkToken:     "link-token-123",
				Expiration:    now.Add(time.Hour),
				RequestID:     "request-123",
				HostedLinkUrl: "https://example.com/hosted-link",
			}

			m.EXPECT().CreateLinkToken(gomock.Any(), client.CreateLinkTokenRequest{
				UserName:       req.PaymentServiceUser.Name,
				UserID:         req.PaymentServiceUser.ID.String(),
				Language:       "en",
				CountryCode:    *req.PaymentServiceUser.Address.Country,
				RedirectURI:    req.RedirectURI,
				WebhookBaseURL: req.WebhookBaseURL,
			}).Return(linkTokenResp, nil)

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateUserLinkResponse{
				Link: linkTokenResp.HostedLinkUrl,
				TemporaryLinkToken: &models.Token{
					Token:     linkTokenResp.LinkToken,
					ExpiresAt: linkTokenResp.Expiration,
				},
			}))
		})
	})
})
