package tink

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Tink *Plugin Delete User", func() {
	Context("delete user", func() {
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

		It("should delete user successfully", func(ctx SpecContext) {
			userID := uuid.New()

			req := models.DeleteUserRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: userID,
				},
			}

			expectedRequest := client.DeleteUserRequest{
				UserID: userID.String(),
			}

			m.EXPECT().DeleteUser(gomock.Any(), expectedRequest).Return(nil)

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})

		It("should return error when client delete user fails", func(ctx SpecContext) {
			userID := uuid.New()

			req := models.DeleteUserRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: userID,
				},
			}

			expectedRequest := client.DeleteUserRequest{
				UserID: userID.String(),
			}

			m.EXPECT().DeleteUser(gomock.Any(), expectedRequest).Return(
				errors.New("client error"),
			)

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.DeleteUserResponse{}))
		})
	})
})
