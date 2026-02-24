package powens

import (
	"errors"

	"github.com/formancehq/payments/pkg/connectors/powens/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin Delete User", func() {
	Context("delete user", func() {
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

		It("should return an error - missing open banking connections", func(ctx SpecContext) {
			req := connector.DeleteUserRequest{}

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("open banking forwarded user is required"))
			Expect(resp).To(Equal(connector.DeleteUserResponse{}))
		})

		It("should return an error - missing auth token", func(ctx SpecContext) {
			req := connector.DeleteUserRequest{
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{},
			}

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("auth token is required"))
			Expect(resp).To(Equal(connector.DeleteUserResponse{}))
		})

		It("should delete user successfully", func(ctx SpecContext) {
			req := connector.DeleteUserRequest{
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					AccessToken: &connector.Token{
						Token: "auth-token-123",
					},
				},
			}

			m.EXPECT().DeleteUser(gomock.Any(), client.DeleteUserRequest{
				AccessToken: "auth-token-123",
			}).Return(nil)

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(connector.DeleteUserResponse{}))
		})

		It("should return an error - client delete user error", func(ctx SpecContext) {
			req := connector.DeleteUserRequest{
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					AccessToken: &connector.Token{
						Token: "auth-token-123",
					},
				},
			}

			m.EXPECT().DeleteUser(gomock.Any(), client.DeleteUserRequest{
				AccessToken: "auth-token-123",
			}).Return(errors.New("client error"))

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(connector.DeleteUserResponse{}))
		})
	})
})
