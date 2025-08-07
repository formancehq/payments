package powens

import (
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin Create User Link", func() {
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
				clientID: "client-123",
				config: Config{
					Domain:         "test.com",
					MaxConnections: 5,
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - missing payment service user", func(ctx SpecContext) {
			req := models.CreateUserLinkRequest{}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("payment service user is required"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return an error - missing bank bridge connections", func(ctx SpecContext) {
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("bank bridge connections are required"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return an error - missing auth token", func(ctx SpecContext) {
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				PSUBankBridge:      &models.PSUBankBridge{},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("auth token is required"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return an error - missing callBackState", func(ctx SpecContext) {
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				PSUBankBridge: &models.PSUBankBridge{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("callBackState is required"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return an error - missing formanceRedirectURL", func(ctx SpecContext) {
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				PSUBankBridge: &models.PSUBankBridge{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
				CallBackState: "state-123",
			}

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("formanceRedirectURL is required"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should create user link successfully", func(ctx SpecContext) {
			redirectURL := "https://formance.com/callback"
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				PSUBankBridge: &models.PSUBankBridge{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
				CallBackState:       "state-123",
				FormanceRedirectURL: &redirectURL,
			}

			temporaryCodeResponse := client.CreateTemporaryLinkResponse{
				Code:      "temp-code-123",
				ExpiredIn: 3600,
			}

			m.EXPECT().CreateTemporaryCode(gomock.Any(), client.CreateTemporaryLinkRequest{
				AccessToken: "auth-token-123",
			}).Return(temporaryCodeResponse, nil)

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Link).To(ContainSubstring("https://webview.powens.com/connect"))
			Expect(resp.Link).To(ContainSubstring("domain=test.com"))
			Expect(resp.Link).To(ContainSubstring("client_id=client-123"))
			Expect(resp.Link).To(ContainSubstring("code=temp-code-123"))
			Expect(resp.Link).To(ContainSubstring("state=state-123"))
			Expect(resp.Link).To(ContainSubstring("max_connections=5"))
			Expect(resp.Link).To(ContainSubstring("redirect_uri=" + redirectURL))
			Expect(resp.TemporaryLinkToken).ToNot(BeNil())
			Expect(resp.TemporaryLinkToken.Token).To(Equal("temp-code-123"))
			Expect(resp.TemporaryLinkToken.ExpiresAt).To(BeTemporally("~", time.Now().Add(3600*time.Second), 2*time.Second))
		})

		It("should return an error - client create temporary code error", func(ctx SpecContext) {
			redirectURL := "https://formance.com/callback"
			req := models.CreateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				PSUBankBridge: &models.PSUBankBridge{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
				CallBackState:       "state-123",
				FormanceRedirectURL: &redirectURL,
			}

			m.EXPECT().CreateTemporaryCode(gomock.Any(), client.CreateTemporaryLinkRequest{
				AccessToken: "auth-token-123",
			}).Return(client.CreateTemporaryLinkResponse{}, errors.New("client error"))

			resp, err := plg.CreateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})
	})
})
