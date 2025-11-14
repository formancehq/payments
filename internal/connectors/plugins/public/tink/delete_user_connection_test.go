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

var _ = Describe("Tink *Plugin Delete User Connection", func() {
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

		It("should delete user connection successfully", func(ctx SpecContext) {
			userID := uuid.New()
			connectionID := "connection_123"

			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
				},
				Connection: &models.PSPOpenBankingConnection{
					ConnectionID: connectionID,
				},
			}

			expectedRequest := client.DeleteUserConnectionRequest{
				UserID:        userID.String(),
				Username:      "Test User",
				CredentialsID: connectionID,
			}

			m.EXPECT().DeleteUserConnection(gomock.Any(), expectedRequest).Return(nil)

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when client delete user connection fails", func(ctx SpecContext) {
			userID := uuid.New()
			connectionID := "connection_123"

			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
				},
				Connection: &models.PSPOpenBankingConnection{
					ConnectionID: connectionID,
				},
			}

			expectedRequest := client.DeleteUserConnectionRequest{
				UserID:        userID.String(),
				Username:      "Test User",
				CredentialsID: connectionID,
			}

			m.EXPECT().DeleteUserConnection(gomock.Any(), expectedRequest).Return(
				errors.New("client error"),
			)

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when payment service user is nil", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: nil,
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("paymentServiceUser is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when payment service user name is empty", func(ctx SpecContext) {
			userID := uuid.New()

			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "",
				},
				Connection: &models.PSPOpenBankingConnection{
					ConnectionID: "connection_123",
				},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("name is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when connection is nil", func(ctx SpecContext) {
			userID := uuid.New()

			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
				},
				Connection: nil,
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("connection is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when connection ID is empty", func(ctx SpecContext) {
			userID := uuid.New()

			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "Test User",
				},
				Connection: &models.PSPOpenBankingConnection{
					ConnectionID: "",
				},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("connectionID is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})
	})
})
