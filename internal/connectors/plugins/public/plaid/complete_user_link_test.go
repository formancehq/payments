package plaid

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Plaid *Plugin Complete User Link", func() {
	Context("complete user link", func() {
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

		It("should return an error - missing related attempt", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("related attempt is required"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - missing temporary token", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.PSUOpenBankingConnectionAttempt{
					ID: uuid.New(),
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing temporary token"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - missing link token", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.PSUOpenBankingConnectionAttempt{
					ID: uuid.New(),
					TemporaryToken: &models.Token{
						Token: "temp-token-123",
					},
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing link token"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - link token mismatch", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.PSUOpenBankingConnectionAttempt{
					ID: uuid.New(),
					TemporaryToken: &models.Token{
						Token: "temp-token-123",
					},
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						client.LinkTokenQueryParamID: {"different-token"},
					},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("link token mismatch"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - missing query values", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.PSUOpenBankingConnectionAttempt{
					ID: uuid.New(),
					TemporaryToken: &models.Token{
						Token: "temp-token-123",
					},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing link token: invalid request"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - missing public token", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.PSUOpenBankingConnectionAttempt{
					ID: uuid.New(),
					TemporaryToken: &models.Token{
						Token: "temp-token-123",
					},
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						client.LinkTokenQueryParamID: {"temp-token-123"},
					},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing public token"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should complete user link successfully", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.PSUOpenBankingConnectionAttempt{
					ID: uuid.New(),
					TemporaryToken: &models.Token{
						Token: "temp-token-123",
					},
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						client.LinkTokenQueryParamID:   {"temp-token-123"},
						client.PublicTokenQueryParamID: {"public-token-123"},
					},
				},
			}

			expectedReq := client.ExchangePublicTokenRequest{
				PublicToken: "public-token-123",
			}

			expectedResp := client.ExchangePublicTokenResponse{
				AccessToken: "access-token-123",
				ItemID:      "item-id-123",
			}

			m.EXPECT().ExchangePublicToken(gomock.Any(), expectedReq).Return(expectedResp, nil)

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Success).ToNot(BeNil())
			Expect(resp.Success.Connections).To(HaveLen(1))
			Expect(resp.Success.Connections[0].ConnectionID).To(Equal("item-id-123"))
			Expect(resp.Success.Connections[0].AccessToken.Token).To(Equal("access-token-123"))
		})

		It("should return an error - client exchange public token error", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.PSUOpenBankingConnectionAttempt{
					ID: uuid.New(),
					TemporaryToken: &models.Token{
						Token: "temp-token-123",
					},
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						client.LinkTokenQueryParamID:   {"temp-token-123"},
						client.PublicTokenQueryParamID: {"public-token-123"},
					},
				},
			}

			m.EXPECT().ExchangePublicToken(gomock.Any(), gomock.Any()).Return(client.ExchangePublicTokenResponse{}, errors.New("client error"))

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})
	})
})
