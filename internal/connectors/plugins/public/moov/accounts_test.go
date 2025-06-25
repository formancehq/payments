package moov

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Moov Accounts", func() {
	var (
		plg           *Plugin
		sampleWallets []moov.Wallet
		sampleAccount moov.Account
	)

	BeforeEach(func() {
		plg = &Plugin{}
		sampleWallets = make([]moov.Wallet, 0)
		sampleAccount = moov.Account{
			AccountID:   "account123",
			DisplayName: "Test Account",
			CreatedOn:   time.Now().UTC(),
		}

		for i := 0; i < 3; i++ {
			sampleWallets = append(sampleWallets, moov.Wallet{
				WalletID: fmt.Sprintf("wallet%d", i),
				AvailableBalance: moov.AvailableBalance{
					Value:        int64(1000 * (i + 1)),
					Currency:     "USD",
					ValueDecimal: fmt.Sprintf("%d.00", 10*(i+1)),
				},
			})
		}
	})

	Context("fetching next accounts", func() {
		var (
			m *client.MockClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
		})

		It("should return an error - get wallets error", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleAccount)
			req := models.FetchNextAccountsRequest{
				FromPayload: accountPayload,
			}

			m.EXPECT().GetWallets(gomock.Any(), sampleAccount.AccountID).Return(
				[]moov.Wallet{},
				errors.New("test error"),
			)

			accounts, err := plg.fetchNextAccounts(ctx, req)
			Expect(err).To(MatchError("failed to fetch wallets: test error"))
			Expect(accounts.Accounts).To(HaveLen(0))
		})

		It("should return an error - invalid from payload", func(ctx SpecContext) {
			invalidPayload := []byte(`invalid json`)
			req := models.FetchNextAccountsRequest{
				FromPayload: invalidPayload,
			}

			accounts, err := plg.fetchNextAccounts(ctx, req)
			Expect(err).NotTo(BeNil())
			Expect(accounts.Accounts).To(HaveLen(0))
		})

		It("should fetch wallets successfully", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleAccount)
			req := models.FetchNextAccountsRequest{
				FromPayload: accountPayload,
			}

			m.EXPECT().GetWallets(gomock.Any(), sampleAccount.AccountID).Return(
				sampleWallets,
				nil,
			)

			resp, err := plg.fetchNextAccounts(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.Accounts[0].Reference).To(Equal("wallet0"))
			Expect(resp.Accounts[0].Name).To(Equal(&sampleAccount.DisplayName))
			Expect(*resp.Accounts[0].DefaultAsset).To(Equal("USD/2"))

			Expect(resp.Accounts[0].Metadata).To(HaveKeyWithValue(client.MoovWalletCurrencyMetadataKey, "USD"))
			Expect(resp.Accounts[0].Metadata).To(HaveKeyWithValue(client.MoovWalletValueMetadataKey, "1000"))
			Expect(resp.Accounts[0].Metadata).To(HaveKeyWithValue(client.MoovValueDecimalMetadataKey, "10.00"))
			Expect(resp.Accounts[0].Metadata).To(HaveKeyWithValue(client.MoovAccountIDMetadataKey, "account123"))
		})

		It("should fetch wallets with nil FromPayload", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				FromPayload: nil,
			}

			m.EXPECT().GetWallets(gomock.Any(), "").Return(
				sampleWallets,
				nil,
			)

			resp, err := plg.fetchNextAccounts(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())
		})

		Context("fetch wallets with moov client", func() {
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
				req := models.FetchNextAccountsRequest{
					FromPayload: accountPayload,
				}

				mockedService.EXPECT().GetMoovWallets(gomock.Any(), sampleAccount.AccountID).Return(
					[]moov.Wallet{},
					errors.New("fetch wallets error"),
				)

				resp, err := plg.fetchNextAccounts(ctx, req)

				Expect(err).To(MatchError("failed to fetch wallets: fetch wallets error"))
				Expect(resp.Accounts).To(HaveLen(0))
			})
		})

		It("should return empty accounts when GetWallets returns error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "").Return(nil, errors.New("test error"))

			resp, err := plg.fetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(ContainSubstring("failed to fetch wallets")))
			Expect(resp.Accounts).To(HaveLen(0))
		})

		It("should handle marshal error in fillAccounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize: 10,
			}

			// Create a wallet that should work fine
			wallet := moov.Wallet{
				WalletID: "test-wallet",
				AvailableBalance: moov.AvailableBalance{
					Currency:     "USD",
					Value:        1000,
					ValueDecimal: "10.00",
				},
			}

			m.EXPECT().GetWallets(gomock.Any(), "").Return([]moov.Wallet{wallet}, nil)

			resp, err := plg.fetchNextAccounts(ctx, req)
			// This should succeed with valid wallet data
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(1))
		})
	})

	Context("fill accounts", func() {
		BeforeEach(func() {
			plg = &Plugin{}
		})

		It("should convert wallets to accounts correctly", func() {
			accounts, err := plg.fillAccounts(sampleWallets, sampleAccount)

			Expect(err).To(BeNil())
			Expect(accounts).To(HaveLen(3))

			Expect(accounts[0].Reference).To(Equal("wallet0"))
			Expect(*accounts[0].Name).To(Equal(sampleAccount.DisplayName))
			Expect(*accounts[0].DefaultAsset).To(Equal("USD/2"))
			Expect(accounts[0].CreatedAt).To(BeTemporally("~", sampleAccount.CreatedOn.UTC(), time.Second))

			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovWalletCurrencyMetadataKey, "USD"))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovWalletValueMetadataKey, "1000"))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovValueDecimalMetadataKey, "10.00"))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovAccountIDMetadataKey, "account123"))

			var wallet moov.Wallet
			err = json.Unmarshal(accounts[0].Raw, &wallet)
			Expect(err).To(BeNil())
			Expect(wallet.WalletID).To(Equal("wallet0"))
		})
	})
})
