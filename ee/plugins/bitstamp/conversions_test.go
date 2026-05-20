package bitstamp

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Bitstamp Plugin Conversions", func() {
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
			logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			currencies: map[string]int{
				"USD":  2,
				"EUR":  2,
				"USDC": 6,
				"BTC":  8,
			},
			currLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	mkTx := func(raw string) client.UserTransaction {
		var tx client.UserTransaction
		Expect(json.Unmarshal([]byte(raw), &tx)).To(Succeed())
		return tx
	}

	Context("fetching next conversions", func() {
		It("returns the wrapped client error when user_transactions fails", func(ctx SpecContext) {
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).
				Return(nil, errors.New("boom"))

			resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{PageSize: 100})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("boom"))
			Expect(err.Error()).To(ContainSubstring("fetch conversions"))
			Expect(resp).To(Equal(models.FetchNextConversionsResponse{}))
		})

		It("emits a PSPConversion for the Quentin #679 EUR -> USDC fixture", func(ctx SpecContext) {
			tx := mkTx(`{
				"id": 458254264,
				"datetime": "2025-09-25 14:42:59.894846",
				"type": "36",
				"fee": "0.000000",
				"eur": "-5.00",
				"usdc": "5.810770",
				"usdc_eur": 0.86047,
				"usd": "0.00",
				"btc": "0.00000000"
			}`)
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).
				Return([]client.UserTransaction{tx}, nil)

			resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{PageSize: 100})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Conversions).To(HaveLen(1))

			c := resp.Conversions[0]
			Expect(c.Reference).To(Equal("458254264"))
			Expect(c.SourceAsset).To(Equal("EUR/2"))
			Expect(c.DestinationAsset).To(Equal("USDC/6"))
			Expect(c.SourceAmount.Int64()).To(Equal(int64(500)))
			Expect(c.DestinationAmount.Int64()).To(Equal(int64(5810770)))
			Expect(c.Status).To(Equal(models.CONVERSION_STATUS_COMPLETED))
			Expect(c.Metadata[mappers.MetadataKeyRate]).To(Equal("0.86047"))
		})

		It("skips type-36 rows that are not two-asset (defensive)", func(ctx SpecContext) {
			tx := mkTx(`{
				"id": 500,
				"datetime": "2025-09-25 14:42:59.000000",
				"type": "36",
				"fee": "0",
				"eur": "-5.00",
				"usdc": "0"
			}`)
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).
				Return([]client.UserTransaction{tx}, nil)

			resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{PageSize: 100})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Conversions).To(BeEmpty())
		})

		It("filters non-type-36 rows from the same stream", func(ctx SpecContext) {
			deposit := mkTx(`{
				"id": 600,
				"datetime": "2025-09-25 14:42:59.000000",
				"type": "0",
				"fee": "0",
				"btc": "1.0"
			}`)
			conv := mkTx(`{
				"id": 601,
				"datetime": "2025-09-25 14:42:59.000000",
				"type": "36",
				"fee": "0.01",
				"eur": "-100.00",
				"usdc": "116.21",
				"usdc_eur": "0.86047"
			}`)
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).
				Return([]client.UserTransaction{deposit, conv}, nil)

			resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{PageSize: 100})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Conversions).To(HaveLen(1))
			Expect(resp.Conversions[0].Reference).To(Equal("601"))
		})

		It("advances the since_id watermark to the max ID seen", func(ctx SpecContext) {
			tx1 := mkTx(`{"id": 700, "datetime": "2025-09-25 14:42:59.000000", "type": "0", "fee": "0", "btc": "0.1"}`)
			tx2 := mkTx(`{"id": 800, "datetime": "2025-09-25 14:42:59.000000", "type": "0", "fee": "0", "btc": "0.1"}`)
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).
				Return([]client.UserTransaction{tx1, tx2}, nil)

			resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{PageSize: 100})
			Expect(err).ToNot(HaveOccurred())

			var state conversionsState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.LastTransactionID).To(Equal(int64(800)))
		})

		It("never resets the watermark on an empty cycle", func(ctx SpecContext) {
			prev := conversionsState{LastTransactionID: 999}
			rawState, _ := json.Marshal(prev)
			pinnedSince := prev.LastTransactionID
			m.EXPECT().GetUserTransactions(gomock.Any(), &pinnedSince, 100).
				Return([]client.UserTransaction{}, nil)

			resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{
				State:    rawState,
				PageSize: 100,
			})
			Expect(err).ToNot(HaveOccurred())

			var state conversionsState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.LastTransactionID).To(Equal(int64(999)))
		})

		It("guards against PageSize <= 0", func(ctx SpecContext) {
			// Limit defaults to PAGE_SIZE when the caller passes 0 so
			// an empty cycle never makes HasMore=true.
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), PAGE_SIZE).
				Return([]client.UserTransaction{}, nil)

			resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{PageSize: 0})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HasMore).To(BeFalse())
		})

		It("Warn-skips derivatives-marked type-36 rows", func(ctx SpecContext) {
			tx := mkTx(`{
				"id": 900,
				"datetime": "2025-09-25 14:42:59.000000",
				"type": "36",
				"fee": "0",
				"eur": "-5.00",
				"usdc": "5.81",
				"margin_mode": "FLEXIBLE"
			}`)
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).
				Return([]client.UserTransaction{tx}, nil)

			resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{PageSize: 100})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Conversions).To(BeEmpty())
		})
	})
})
