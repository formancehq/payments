package plaid

import (
	"errors"

	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Plaid *Plugin Delete User Connection", func() {
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

		It("should return an error - missing access token", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{
				Connection: &connector.PSPOpenBankingConnection{
					ConnectionID: "test-connection",
				},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("access token is required"))
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})

		It("should delete user connection successfully", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{
				Connection: &connector.PSPOpenBankingConnection{
					ConnectionID: "test-connection",
					AccessToken: &connector.Token{
						Token: "access-token-123",
					},
				},
			}

			expectedReq := client.DeleteItemRequest{
				AccessToken: "access-token-123",
			}

			m.EXPECT().DeleteItem(gomock.Any(), expectedReq).Return(nil)

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})

		It("should return an error - client delete item error", func(ctx SpecContext) {
			req := connector.DeleteUserConnectionRequest{
				Connection: &connector.PSPOpenBankingConnection{
					ConnectionID: "test-connection",
					AccessToken: &connector.Token{
						Token: "access-token-123",
					},
				},
			}

			m.EXPECT().DeleteItem(gomock.Any(), gomock.Any()).Return(errors.New("client error"))

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to delete item"))
			Expect(resp).To(Equal(connector.DeleteUserConnectionResponse{}))
		})
	})
})
