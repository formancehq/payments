package plaid

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Plaid *Plugin Delete User Connection", func() {
	Context("delete user connection", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
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
			req := models.DeleteUserConnectionRequest{}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("connection is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return an error - missing access token", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSPOpenBankingConnection{
					ConnectionID: "test-connection",
				},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("access token is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should delete user connection successfully", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSPOpenBankingConnection{
					ConnectionID: "test-connection",
					AccessToken: &models.Token{
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
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return an error - client delete item error", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSPOpenBankingConnection{
					ConnectionID: "test-connection",
					AccessToken: &models.Token{
						Token: "access-token-123",
					},
				},
			}

			m.EXPECT().DeleteItem(gomock.Any(), gomock.Any()).Return(errors.New("client error"))

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to delete item"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})
	})
})
