package moov

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Moov Balances", func() {
	var (
		plg           *Plugin
		sampleWallet  *moov.Wallet
		sampleAccount models.PSPAccount
	)

	BeforeEach(func() {
		plg = &Plugin{}
		sampleWallet = &moov.Wallet{
			WalletID: "wallet123",
			AvailableBalance: moov.AvailableBalance{
				Value:        int64(1000),
				Currency:     "USD",
				ValueDecimal: "10.00",
			},
		}

		sampleAccount = models.PSPAccount{
			Reference: "wallet123",
			Metadata: map[string]string{
				client.MoovAccountIDMetadataKey: "account123",
			},
		}
	})

	Context("fetching next balances", func() {
		var (
			m *client.MockClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
		})

		It("should return ErrNotYetInstalled when client is nil", func(ctx SpecContext) {
			// Setting client to nil to trigger the error
			plg.client = nil

			req := models.FetchNextBalancesRequest{
				FromPayload: nil,
			}

			resp, err := plg.FetchNextBalances(ctx, req)

			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.Balances).To(HaveLen(0))
		})

		It("should return an error - get wallet error", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleAccount)
			req := models.FetchNextBalancesRequest{
				FromPayload: accountPayload,
			}

			m.EXPECT().GetWallet(gomock.Any(), "account123", "wallet123").Return(
				nil,
				errors.New("test error"),
			)

			balances, err := plg.fetchNextBalances(ctx, req)
			Expect(err).To(MatchError("failed to fetch wallet: test error"))
			Expect(balances.Balances).To(HaveLen(0))
		})

		It("should return an error - invalid from payload", func(ctx SpecContext) {
			invalidPayload := []byte(`invalid json`)
			req := models.FetchNextBalancesRequest{
				FromPayload: invalidPayload,
			}

			balances, err := plg.fetchNextBalances(ctx, req)
			Expect(err).NotTo(BeNil())
			Expect(balances.Balances).To(HaveLen(0))
		})

		It("should fetch wallet balance successfully", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleAccount)
			req := models.FetchNextBalancesRequest{
				FromPayload: accountPayload,
			}

			m.EXPECT().GetWallet(gomock.Any(), "account123", "wallet123").Return(
				sampleWallet,
				nil,
			)

			resp, err := plg.fetchNextBalances(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.Balances[0].AccountReference).To(Equal("wallet123"))
			Expect(resp.Balances[0].Asset).To(Equal("USD/2"))
			Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(1000)))
			Expect(resp.Balances[0].CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should fetch wallet balance with nil FromPayload", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				FromPayload: nil,
			}

			m.EXPECT().GetWallet(gomock.Any(), "", "").Return(
				sampleWallet,
				nil,
			)

			resp, err := plg.fetchNextBalances(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
		})

		Context("fetch wallet balance with moov client", func() {
			var (
				mockedService *client.MockMoovClient
			)

			BeforeEach((func() {
				ctrl := gomock.NewController(GinkgoT())
				mockedService = client.NewMockMoovClient(ctrl)

				plg.client, _ = client.New("moov", "https://example.com", "access_token", "test", "test")
				plg.client.NewWithClient(mockedService)
			}))

			It("should fail when moov client returns an error", func(ctx SpecContext) {
				accountPayload, _ := json.Marshal(sampleAccount)
				req := models.FetchNextBalancesRequest{
					FromPayload: accountPayload,
				}

				mockedService.EXPECT().GetMoovWallet(gomock.Any(), "account123", "wallet123").Return(
					nil,
					errors.New("fetch wallet error"),
				)

				resp, err := plg.fetchNextBalances(ctx, req)

				Expect(err).To(MatchError("failed to fetch wallet: fetch wallet error"))
				Expect(resp.Balances).To(HaveLen(0))
			})

			It("should successfully fetch balance using moov client", func(ctx SpecContext) {
				accountPayload, _ := json.Marshal(sampleAccount)
				req := models.FetchNextBalancesRequest{
					FromPayload: accountPayload,
				}

				mockedService.EXPECT().GetMoovWallet(gomock.Any(), "account123", "wallet123").Return(
					sampleWallet,
					nil,
				)

				resp, err := plg.fetchNextBalances(ctx, req)

				Expect(err).To(BeNil())
				Expect(resp.Balances).To(HaveLen(1))
				Expect(resp.HasMore).To(BeFalse())
				Expect(resp.Balances[0].AccountReference).To(Equal("wallet123"))
				Expect(resp.Balances[0].Asset).To(Equal("USD/2"))
				Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(1000)))
				Expect(resp.Balances[0].CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
			})
		})
	})
})
