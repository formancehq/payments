package plaid

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Plaid *Plugin Create User", func() {
	Context("create user", func() {
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

		It("should return an error - missing payment service user", func(ctx SpecContext) {
			req := models.CreateUserRequest{}

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("payment service user is required"))
			Expect(resp).To(Equal(models.CreateUserResponse{}))
		})

		It("should create user successfully", func(ctx SpecContext) {
			userID := uuid.New()
			req := models.CreateUserRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: userID,
				},
			}

			m.EXPECT().CreateUser(gomock.Any(), userID.String()).Return("user-token-123", nil)

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Metadata[UserTokenMetadataKey]).To(Equal("user-token-123"))
		})

		It("should return an error - client create user error", func(ctx SpecContext) {
			userID := uuid.New()
			req := models.CreateUserRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: userID,
				},
			}

			m.EXPECT().CreateUser(gomock.Any(), userID.String()).Return("", errors.New("client error"))

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.CreateUserResponse{}))
		})
	})
})
