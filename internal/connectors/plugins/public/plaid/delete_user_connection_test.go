package plaid

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Plaid *Plugin Delete User Connection", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("delete user connection", func() {
		var (
			sampleRequest models.DeleteUserConnectionRequest
		)

		BeforeEach(func() {
			sampleRequest = models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				},
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: "access-token-123",
				},
			}
		})

		It("should return error - validation error - missing payment service user", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("payment service user is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error - validation error - missing bank bridge consent", func(ctx SpecContext) {
			req := sampleRequest
			req.BankBridgeConsent = nil

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("bank bridge consent is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error - validation error - missing access token", func(ctx SpecContext) {
			req := sampleRequest
			req.BankBridgeConsent.AccessToken = ""

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("access token is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error - delete item error", func(ctx SpecContext) {
			req := sampleRequest

			m.EXPECT().DeleteItem(gomock.Any(), client.DeleteItemRequest{
				AccessToken: req.BankBridgeConsent.AccessToken,
			}).Return(errors.New("test error"))

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to delete item"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := sampleRequest

			m.EXPECT().DeleteItem(gomock.Any(), client.DeleteItemRequest{
				AccessToken: req.BankBridgeConsent.AccessToken,
			}).Return(nil)

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})
	})
})
