package powens

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin CreateUserLink", func() {
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

	Context("create user link", func() {
		It("should return valid user link response", func(ctx SpecContext) {
			// Mock CreateUser response
			createUserResp := client.CreateUserResponse{
				AuthToken: "test-auth-token",
				Type:      "user",
				IdUser:    123,
				ExpiresIn: 3600,
			}
			m.EXPECT().CreateUser(gomock.Any()).Return(createUserResp, nil)

			// Mock CreateTemporaryLink response
			createTempLinkResp := client.CreateTemporaryLinkResponse{
				Code:      "test-code",
				Type:      "temporary",
				Access:    "read",
				ExpiredIn: 300,
			}
			m.EXPECT().CreateTemporaryLink(gomock.Any(), client.CreateTemporaryLinkRequest{
				AccessToken: "test-auth-token",
				RedirectURI: "https://example.com/callback",
			}).Return(createTempLinkResp, nil)

			req := models.CreateUserLinkRequest{
				RedirectURI: "https://example.com/callback",
			}

			resp, err := plg.createUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Link).To(Equal("https://webview.powens.com/connect?domain=formance-sandbox&client_id=test-client-id&redirect_uri=https://example.com/callback&code=test-code"))
			Expect(resp.TemporaryLinkToken).ToNot(BeNil())
			Expect(resp.TemporaryLinkToken.Token).To(Equal("test-code"))
			Expect(resp.TemporaryLinkToken.ExpiresAt).To(BeTemporally("~", time.Now().Add(300*time.Second), time.Second))
			Expect(resp.PermanentToken).ToNot(BeNil())
			Expect(resp.PermanentToken.Token).To(Equal("test-auth-token"))
		})

		It("should return error when CreateUser fails", func(ctx SpecContext) {
			m.EXPECT().CreateUser(gomock.Any()).Return(client.CreateUserResponse{}, fmt.Errorf("test error"))

			req := models.CreateUserLinkRequest{
				RedirectURI: "https://example.com/callback",
			}

			resp, err := plg.createUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("test error"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})

		It("should return error when CreateTemporaryLink fails", func(ctx SpecContext) {
			// Mock CreateUser response
			createUserResp := client.CreateUserResponse{
				AuthToken: "test-auth-token",
				Type:      "user",
				IdUser:    123,
				ExpiresIn: 3600,
			}
			m.EXPECT().CreateUser(gomock.Any()).Return(createUserResp, nil)

			// Mock CreateTemporaryLink failure
			m.EXPECT().CreateTemporaryLink(gomock.Any(), client.CreateTemporaryLinkRequest{
				AccessToken: "test-auth-token",
				RedirectURI: "https://example.com/callback",
			}).Return(client.CreateTemporaryLinkResponse{}, fmt.Errorf("temporary link error"))

			req := models.CreateUserLinkRequest{
				RedirectURI: "https://example.com/callback",
			}

			resp, err := plg.createUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("temporary link error"))
			Expect(resp).To(Equal(models.CreateUserLinkResponse{}))
		})
	})
})
