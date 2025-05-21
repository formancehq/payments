package powens

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin DeleteUserConnection", func() {
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

	Context("delete user connection", func() {
		It("should successfully delete user connection", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: "test-access-token",
				},
				Connection: &models.PSUBankBridgeConnection{
					ConnectionID: "test-connection-id",
				},
			}

			m.EXPECT().DeleteUserConnection(gomock.Any(), client.DeleteUserConnectionRequest{
				AccessToken:  "test-access-token",
				ConnectionID: "test-connection-id",
			}).Return(nil)

			resp, err := plg.deleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when bank bridge consent is missing", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSUBankBridgeConnection{
					ConnectionID: "test-connection-id",
				},
			}

			resp, err := plg.deleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("bank bridge consent is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when access token is missing", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				BankBridgeConsent: &models.PSUBankBridgeConsent{},
				Connection: &models.PSUBankBridgeConnection{
					ConnectionID: "test-connection-id",
				},
			}

			resp, err := plg.deleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("access token is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when connection is missing", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: "test-access-token",
				},
			}

			resp, err := plg.deleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("connection is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when connection ID is missing", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: "test-access-token",
				},
				Connection: &models.PSUBankBridgeConnection{},
			}

			resp, err := plg.deleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("connection id is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when client delete fails", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: "test-access-token",
				},
				Connection: &models.PSUBankBridgeConnection{
					ConnectionID: "test-connection-id",
				},
			}

			m.EXPECT().DeleteUserConnection(gomock.Any(), client.DeleteUserConnectionRequest{
				AccessToken:  "test-access-token",
				ConnectionID: "test-connection-id",
			}).Return(fmt.Errorf("delete error"))

			resp, err := plg.deleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("delete error"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})
	})
})
