package coinbaseprime

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/coinbase-samples/prime-sdk-go/model"
	"github.com/coinbase-samples/prime-sdk-go/wallets"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Prime Plugin Accounts", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetch next accounts", func() {
		var (
			pageSize      int
			sampleWallets []*model.Wallet
		)

		BeforeEach(func() {
			pageSize = 20

			sampleWallets = make([]*model.Wallet, 0)
			for i := 0; i < pageSize; i++ {
				sampleWallets = append(sampleWallets, &model.Wallet{
					Id:      fmt.Sprintf("wallet-%d", i),
					Name:    fmt.Sprintf("Wallet %d", i),
					Symbol:  "BTC",
					Type:    "VAULT",
					Created: time.Now(),
				})
			}
		})

		It("fetches next accounts successfully", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: pageSize,
			}

			m.EXPECT().GetWallets(gomock.Any(), "", pageSize).Return(
				&wallets.ListWalletsResponse{
					Wallets: sampleWallets,
					Pagination: &model.Pagination{
						NextCursor: "next-cursor",
						HasNext:    true,
					},
				},
				nil,
			)

			res, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.Accounts).To(HaveLen(pageSize))
			Expect(res.Accounts[0].Reference).To(Equal(sampleWallets[0].Id))

			var state accountsState
			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Cursor).To(Equal("next-cursor"))
		})

		It("fetches next accounts with cursor", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    json.RawMessage(`{"cursor": "previous-cursor"}`),
				PageSize: pageSize,
			}

			m.EXPECT().GetWallets(gomock.Any(), "previous-cursor", pageSize).Return(
				&wallets.ListWalletsResponse{
					Wallets: sampleWallets[:5],
					Pagination: &model.Pagination{
						NextCursor: "",
						HasNext:    false,
					},
				},
				nil,
			)

			res, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Accounts).To(HaveLen(5))
		})

		It("returns error on client failure", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: pageSize,
			}

			m.EXPECT().GetWallets(gomock.Any(), "", pageSize).Return(
				nil,
				fmt.Errorf("client error"),
			)

			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get wallets"))
		})

		It("returns error on invalid state", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    json.RawMessage(`{invalid json}`),
				PageSize: pageSize,
			}

			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal state"))
		})
	})
})
