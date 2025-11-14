package powens

import (
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Powens *Plugin Update User Link", func() {
	Context("update user link", func() {
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
					Domain: "test.com",
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - missing payment service user", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("payment service user is required"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing open banking connections", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				Connection: &models.OpenBankingConnection{
					ConnectionID: "test",
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("open banking connections are required"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing auth token", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				PaymentServiceUser:       &models.PSPPaymentServiceUser{},
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{},
				Connection: &models.OpenBankingConnection{
					ConnectionID: "test",
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("auth token is required"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing callBackState", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
				Connection: &models.OpenBankingConnection{
					ConnectionID: "test",
				},
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("callBackState is required"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should return an error - missing formanceRedirectURL", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
				Connection: &models.OpenBankingConnection{
					ConnectionID: "test",
				},
				CallBackState: "state-123",
			}

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("formanceRedirectURL is required"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})

		It("should update user link successfully", func(ctx SpecContext) {
			redirectURL := "https://formance.com/callback"
			req := models.UpdateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
				Connection: &models.OpenBankingConnection{
					ConnectionID: "conn-123",
				},
				CallBackState:       "state-123",
				FormanceRedirectURL: &redirectURL,
			}

			temporaryCodeResponse := client.CreateTemporaryLinkResponse{
				Code:      "temp-code-123",
				ExpiresIn: 3600,
			}

			m.EXPECT().CreateTemporaryCode(gomock.Any(), client.CreateTemporaryLinkRequest{
				AccessToken: "auth-token-123",
			}).Return(temporaryCodeResponse, nil)

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Link).To(ContainSubstring(powensWebviewBaseURL + "/reconnect"))
			Expect(resp.Link).To(ContainSubstring("domain=test.com"))
			Expect(resp.Link).To(ContainSubstring("client_id=client-123"))
			Expect(resp.Link).To(ContainSubstring("code=temp-code-123"))
			Expect(resp.Link).To(ContainSubstring("connection_id=conn-123"))
			Expect(resp.Link).To(ContainSubstring("state=state-123"))
			Expect(resp.Link).To(ContainSubstring("redirect_uri=" + redirectURL))
			Expect(resp.TemporaryLinkToken).ToNot(BeNil())
			Expect(resp.TemporaryLinkToken.Token).To(Equal("temp-code-123"))
			Expect(resp.TemporaryLinkToken.ExpiresAt).To(BeTemporally("~", time.Now().Add(3600*time.Second), 2*time.Second))
		})

		It("should return an error - client create temporary code error", func(ctx SpecContext) {
			redirectURL := "https://formance.com/callback"
			req := models.UpdateUserLinkRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{},
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
				Connection: &models.OpenBankingConnection{
					ConnectionID: "conn-123",
				},
				CallBackState:       "state-123",
				FormanceRedirectURL: &redirectURL,
			}

			m.EXPECT().CreateTemporaryCode(gomock.Any(), client.CreateTemporaryLinkRequest{
				AccessToken: "auth-token-123",
			}).Return(client.CreateTemporaryLinkResponse{}, errors.New("client error"))

			resp, err := plg.UpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.UpdateUserLinkResponse{}))
		})
	})
})
