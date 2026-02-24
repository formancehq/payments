package powens

import (
	"errors"

	"github.com/formancehq/payments/pkg/connectors/powens/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin Delete User Connection", func() {
	Context("delete user connection", func() {
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

		It("should return an error - missing connection", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("connection is required"))
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})

		It("should return an error - missing connection id", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{
				Connection: &connector.PSPOpenBankingConnection{},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("connection id is required"))
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})

		It("should return an error - missing open banking connections", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{
				Connection: &connector.PSPOpenBankingConnection{
					ConnectionID: "conn-123",
				},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("open banking forwarded user is required"))
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})

		It("should return an error - missing auth token", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{
				Connection: &connector.PSPOpenBankingConnection{
					ConnectionID: "conn-123",
				},
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("auth token is required"))
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})

		It("should delete user connection successfully", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{
				Connection: &connector.PSPOpenBankingConnection{
					ConnectionID: "conn-123",
				},
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					AccessToken: &connector.Token{
						Token: "auth-token-123",
					},
				},
			}

			m.EXPECT().DeleteUserConnection(gomock.Any(), client.DeleteUserConnectionRequest{
				AccessToken:  "auth-token-123",
				ConnectionID: "conn-123",
			}).Return(nil)

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})

		It("should return an error - client delete user connection error", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{
				Connection: &connector.PSPOpenBankingConnection{
					ConnectionID: "conn-123",
				},
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					AccessToken: &connector.Token{
						Token: "auth-token-123",
					},
				},
			}

			m.EXPECT().DeleteUserConnection(gomock.Any(), client.DeleteUserConnectionRequest{
				AccessToken:  "auth-token-123",
				ConnectionID: "conn-123",
			}).Return(errors.New("client error"))

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})
	})
})
