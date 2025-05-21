package tink

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Delete User", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			client: m,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("DeleteUser", func() {
		It("should return error when plugin is not installed", func(ctx SpecContext) {
			plg.client = nil
			resp, err := plg.DeleteUser(ctx, models.DeleteUserRequest{})
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})

		It("should return error when payment service user is missing", func(ctx SpecContext) {
			resp, err := plg.DeleteUser(ctx, models.DeleteUserRequest{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("paymentServiceUser is required"))
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})

		It("should successfully delete user", func(ctx SpecContext) {
			userID := uuid.New()
			req := models.DeleteUserRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: userID,
				},
			}

			m.EXPECT().
				DeleteUser(ctx, client.DeleteUserRequest{
					UserID: userID.String(),
				}).
				Return(nil)

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})

		It("should return error when client returns error", func(ctx SpecContext) {
			userID := uuid.New()
			req := models.DeleteUserRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: userID,
				},
			}

			expectedErr := fmt.Errorf("client error")
			m.EXPECT().
				DeleteUser(ctx, client.DeleteUserRequest{
					UserID: userID.String(),
				}).
				Return(expectedErr)

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).To(Equal(expectedErr))
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})
	})
})
