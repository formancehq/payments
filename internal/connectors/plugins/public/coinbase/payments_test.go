package coinbase

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbase/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Plugin Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			client: m,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next payments", func() {
		var (
			now             time.Time
			completedAt     time.Time
			sampleTransfers []client.Transfer
		)

		BeforeEach(func() {
			now = time.Now().UTC()
			completedAt = now.Add(-time.Hour)

			sampleTransfers = []client.Transfer{
				{
					ID:          "transfer1",
					Type:        "deposit",
					CreatedAt:   now.Add(-2 * time.Hour),
					CompletedAt: &completedAt,
					Amount:      "1.5",
					Currency:    "BTC",
					Details: client.TransferDetails{
						CryptoTransactionHash: "0xabc123",
						CryptoAddress:         "bc1qxyz",
					},
				},
				{
					ID:        "transfer2",
					Type:      "withdraw",
					CreatedAt: now.Add(-time.Hour),
					Amount:    "-500.00",
					Currency:  "USD",
					Details: client.TransferDetails{
						SentToAddress: "bank-account-123",
					},
				},
				{
					ID:        "transfer3",
					Type:      "internal_deposit",
					CreatedAt: now,
					Amount:    "100.00",
					Currency:  "USDC",
				},
			}
		})

		It("should return an error - get transfers error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransfers(gomock.Any(), "", 10).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch payments successfully", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransfers(gomock.Any(), "", 10).Return(
				&client.TransfersResponse{
					Transfers:  sampleTransfers,
					NextCursor: "cursor123",
					HasMore:    true,
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(3))
			Expect(resp.HasMore).To(BeTrue())

			// Verify first payment (BTC deposit)
			Expect(resp.Payments[0].Reference).To(Equal("transfer1"))
			Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(resp.Payments[0].Asset).To(Equal("BTC/8"))
			// 1.5 BTC = 150000000 (1.5 * 10^8)
			Expect(resp.Payments[0].Amount.Cmp(big.NewInt(150000000))).To(Equal(0))
			Expect(resp.Payments[0].Metadata["crypto_transaction_hash"]).To(Equal("0xabc123"))

			// Verify second payment (USD withdrawal)
			Expect(resp.Payments[1].Reference).To(Equal("transfer2"))
			Expect(resp.Payments[1].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			Expect(resp.Payments[1].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
			Expect(resp.Payments[1].Asset).To(Equal("USD/2"))
			// 500.00 USD = 50000 (500.00 * 10^2), negative sign removed
			Expect(resp.Payments[1].Amount.Cmp(big.NewInt(50000))).To(Equal(0))

			// Verify third payment (USDC internal deposit)
			Expect(resp.Payments[2].Reference).To(Equal("transfer3"))
			Expect(resp.Payments[2].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(resp.Payments[2].Asset).To(Equal("USDC/6"))
			// 100.00 USDC = 100000000 (100.00 * 10^6)
			Expect(resp.Payments[2].Amount.Cmp(big.NewInt(100000000))).To(Equal(0))
		})

		It("should skip unsupported currencies", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			unsupportedTransfers := []client.Transfer{
				{
					ID:        "transfer1",
					Type:      "deposit",
					CreatedAt: now,
					Amount:    "100",
					Currency:  "UNKNOWN_CURRENCY",
				},
				{
					ID:        "transfer2",
					Type:      "deposit",
					CreatedAt: now,
					Amount:    "1.0",
					Currency:  "BTC",
				},
			}

			m.EXPECT().GetTransfers(gomock.Any(), "", 10).Return(
				&client.TransfersResponse{
					Transfers:  unsupportedTransfers,
					NextCursor: "",
					HasMore:    false,
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			// Only BTC transfer should be returned
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Reference).To(Equal("transfer2"))
		})

		It("should handle cancelled transfers", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			cancelledAt := now.Add(-time.Hour)
			cancelledTransfers := []client.Transfer{
				{
					ID:         "transfer1",
					Type:       "withdraw",
					CreatedAt:  now.Add(-2 * time.Hour),
					CanceledAt: &cancelledAt,
					Amount:     "100.00",
					Currency:   "USD",
				},
			}

			m.EXPECT().GetTransfers(gomock.Any(), "", 10).Return(
				&client.TransfersResponse{
					Transfers:  cancelledTransfers,
					NextCursor: "",
					HasMore:    false,
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_CANCELLED))
		})

		It("should use cursor for pagination", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{"cursor": "existing-cursor"}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransfers(gomock.Any(), "existing-cursor", 10).Return(
				&client.TransfersResponse{
					Transfers:  []client.Transfer{},
					NextCursor: "",
					HasMore:    false,
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
		})
	})
})
