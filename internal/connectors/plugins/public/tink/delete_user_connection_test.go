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

var _ = Describe("Delete User Connection", func() {
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

	Context("DeleteUserConnection", func() {
		It("should return error when plugin is not installed", func(ctx SpecContext) {
			plg.client = nil
			resp, err := plg.DeleteUserConnection(ctx, models.DeleteUserConnectionRequest{})
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when payment service user is missing", func(ctx SpecContext) {
			resp, err := plg.DeleteUserConnection(ctx, models.DeleteUserConnectionRequest{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("paymentServiceUser is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when payment service user name is missing", func(ctx SpecContext) {
			userID := uuid.New()
			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID: userID,
				},
			}
			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("name is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when bank bridge consent is missing", func(ctx SpecContext) {
			userID := uuid.New()
			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
				},
			}
			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("bankBridgeConsent is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when access token is missing", func(ctx SpecContext) {
			userID := uuid.New()
			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
				},
				BankBridgeConsent: &models.PSUBankBridgeConsent{},
			}
			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("accessToken is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should successfully delete user connection", func(ctx SpecContext) {
			userID := uuid.New()
			accessToken := "test-access-token"
			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
				},
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: accessToken,
				},
			}

			m.EXPECT().
				DeleteUserConnection(ctx, client.DeleteUserConnectionRequest{
					UserID:        userID.String(),
					Username:      "test-user",
					CredentialsID: accessToken,
				}).
				Return(nil)

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return error when client returns error", func(ctx SpecContext) {
			userID := uuid.New()
			accessToken := "test-access-token"
			req := models.DeleteUserConnectionRequest{
				PaymentServiceUser: &models.PSPPaymentServiceUser{
					ID:   userID,
					Name: "test-user",
				},
				BankBridgeConsent: &models.PSUBankBridgeConsent{
					AccessToken: accessToken,
				},
			}

			expectedErr := fmt.Errorf("client error")
			m.EXPECT().
				DeleteUserConnection(ctx, client.DeleteUserConnectionRequest{
					UserID:        userID.String(),
					Username:      "test-user",
					CredentialsID: accessToken,
				}).
				Return(expectedErr)

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).To(Equal(expectedErr))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})
	})
})
