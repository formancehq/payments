package fireblocks

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Fireblocks Plugin Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			logger:         logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			client:         m,
			assetDecimals:  map[string]int{"BTC": 8, "USD": 2},
			assetsLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("fetches next payments and updates state", func(ctx SpecContext) {
		state, err := json.Marshal(paymentsState{LastCreatedAt: 1000, LastTxID: "a"})
		Expect(err).To(BeNil())

		m.EXPECT().ListTransactions(gomock.Any(), int64(1000), 3).Return([]client.Transaction{
			{
				ID:         "a",
				AssetID:    "BTC",
				AmountInfo: client.AmountInfo{Amount: "1"},
				Operation:  "TRANSFER",
				Status:     "COMPLETED",
				CreatedAt:  1000,
			},
			{
				ID:         "b",
				AssetID:    "BTC",
				AmountInfo: client.AmountInfo{Amount: "1"},
				Operation:  "TRANSFER",
				Status:     "COMPLETED",
				CreatedAt:  1000,
				Source:     client.TransferPeer{ID: "src"},
				Destination: client.TransferPeer{
					ID: "dst",
				},
				TxHash: "hash",
				FeeInfo: client.FeeInfo{
					NetworkFee: "0.01",
				},
			},
			{
				ID:         "c",
				AssetID:    "USD",
				AmountInfo: client.AmountInfo{Amount: "10.50"},
				Operation:  "WITHDRAW",
				Status:     "PENDING_SIGNATURE",
				CreatedAt:  1001,
			},
		}, nil)

		resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{
			State:    state,
			PageSize: 3,
		})
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeTrue())
		Expect(resp.Payments).To(HaveLen(2))

		first := resp.Payments[0]
		Expect(first.Reference).To(Equal("b"))
		Expect(first.Amount).To(Equal(big.NewInt(100000000)))
		Expect(first.Asset).To(Equal("BTC/8"))
		Expect(first.Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
		Expect(first.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
		Expect(*first.SourceAccountReference).To(Equal("src"))
		Expect(*first.DestinationAccountReference).To(Equal("dst"))
		Expect(first.Metadata["txHash"]).To(Equal("hash"))
		Expect(first.Metadata["networkFee"]).To(Equal("0.01"))

		second := resp.Payments[1]
		Expect(second.Reference).To(Equal("c"))
		Expect(second.Amount).To(Equal(big.NewInt(1050)))
		Expect(second.Asset).To(Equal("USD/2"))
		Expect(second.Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
		Expect(second.Status).To(Equal(models.PAYMENT_STATUS_PENDING))

		var newState paymentsState
		err = json.Unmarshal(resp.NewState, &newState)
		Expect(err).To(BeNil())
		Expect(newState.LastCreatedAt).To(Equal(int64(1001)))
		Expect(newState.LastTxID).To(Equal("c"))
	})

	It("advances state even when transactions are skipped", func(ctx SpecContext) {
		state, err := json.Marshal(paymentsState{LastCreatedAt: 2000, LastTxID: "z"})
		Expect(err).To(BeNil())

		m.EXPECT().ListTransactions(gomock.Any(), int64(2000), 1).Return([]client.Transaction{
			{
				ID:         "skipped",
				AssetID:    "UNKNOWN",
				AmountInfo: client.AmountInfo{Amount: "1"},
				Operation:  "TRANSFER",
				Status:     "COMPLETED",
				CreatedAt:  2001,
			},
		}, nil)

		resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{
			State:    state,
			PageSize: 1,
		})
		Expect(err).To(BeNil())
		Expect(resp.Payments).To(BeEmpty())

		var newState paymentsState
		err = json.Unmarshal(resp.NewState, &newState)
		Expect(err).To(BeNil())
		Expect(newState.LastCreatedAt).To(Equal(int64(2001)))
		Expect(newState.LastTxID).To(Equal("skipped"))
	})
})
