package powens

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin Delete User Connection", func() {
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

		It("should return an error - missing connection id", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSPPsuOpenBankingConnection{},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("connection id is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return an error - missing open banking connections", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSPPsuOpenBankingConnection{
					ConnectionID: "conn-123",
				},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("open banking provider psu is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return an error - missing auth token", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSPPsuOpenBankingConnection{
					ConnectionID: "conn-123",
				},
				OpenBankingProviderPSU: &models.OpenBankingProviderPSU{},
			}

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("auth token is required"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should delete user connection successfully", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSPPsuOpenBankingConnection{
					ConnectionID: "conn-123",
				},
				OpenBankingProviderPSU: &models.OpenBankingProviderPSU{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
			}

			m.EXPECT().DeleteUserConnection(gomock.Any(), client.DeleteUserConnectionRequest{
				AccessToken:  "auth-token-123",
				ConnectionID: "conn-123",
			}).Return(nil)

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})

		It("should return an error - client delete user connection error", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{
				Connection: &models.PSPPsuOpenBankingConnection{
					ConnectionID: "conn-123",
				},
				OpenBankingProviderPSU: &models.OpenBankingProviderPSU{
					AccessToken: &models.Token{
						Token: "auth-token-123",
					},
				},
			}

			m.EXPECT().DeleteUserConnection(gomock.Any(), client.DeleteUserConnectionRequest{
				AccessToken:  "auth-token-123",
				ConnectionID: "conn-123",
			}).Return(errors.New("client error"))

			resp, err := plg.DeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.DeleteUserConnectionResponse{}))
		})
	})
})
