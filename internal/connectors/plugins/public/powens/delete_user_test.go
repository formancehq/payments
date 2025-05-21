package powens

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin DeleteUser", func() {
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

	Context("delete user", func() {
		It("should successfully delete user", func(ctx SpecContext) {
			req := models.DeleteUserRequest{
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: "test-access-token",
				},
			}

			m.EXPECT().DeleteUser(gomock.Any(), client.DeleteUserRequest{
				AccessToken: "test-access-token",
			}).Return(nil)

			resp, err := plg.deleteUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})

		It("should return error when bank bridge consent is missing", func(ctx SpecContext) {
			req := models.DeleteUserRequest{}

			resp, err := plg.deleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("bank bridge consent is required"))
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})

		It("should return error when access token is missing", func(ctx SpecContext) {
			req := models.DeleteUserRequest{
				BankBridgeConsent: &models.PSUBankBridgeConsent{},
			}

			resp, err := plg.deleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("access token is required"))
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})

		It("should return error when client delete fails", func(ctx SpecContext) {
			req := models.DeleteUserRequest{
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: "test-access-token",
				},
			}

			m.EXPECT().DeleteUser(gomock.Any(), client.DeleteUserRequest{
				AccessToken: "test-access-token",
			}).Return(fmt.Errorf("delete error"))

			resp, err := plg.deleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("delete error"))
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})
	})
})
