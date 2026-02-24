package tink

import (
	"errors"

	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Delete User", func() {
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

		It("should delete user successfully", func(ctx SpecContext) {
			userID := uuid.New()

			req := connector.DeleteUserRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID: userID,
				},
			}

			expectedRequest := client.DeleteUserRequest{
				UserID: userID.String(),
			}

			m.EXPECT().DeleteUser(gomock.Any(), expectedRequest).Return(nil)

			resp, err := plg.DeleteUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(connector.DeleteUserResponse{}))
		})

		It("should return error when client delete user fails", func(ctx SpecContext) {
			userID := uuid.New()

			req := connector.DeleteUserRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
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
			Expect(resp).To(Equal(connector.DeleteUserResponse{}))
		})
	})
})
