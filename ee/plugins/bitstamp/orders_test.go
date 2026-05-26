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

var _ = Describe("Bitstamp Plugin Orders", func() {
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
				"USD": 2,
				"EUR": 2,
				"BTC": 8,
				"ETH": 18,
			},
			currLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	// fromPayloadWithMarkets builds a FetchNextOrdersRequest.FromPayload
	// containing a PSPAccount whose metadata lists the given markets.
	// The engine unwraps its own envelope before calling the plugin, so
	// FromPayload is just the raw PSPAccount JSON.
	fromPayloadWithMarkets := func(markets []string) json.RawMessage {
		marketsRaw, _ := json.Marshal(markets)
		account := models.PSPAccount{
			Reference: "BTC",
			Metadata:  map[string]string{mappers.MetadataKeyTradableMarkets: string(marketsRaw)},
		}
		payload, _ := json.Marshal(account)
		return payload
	}

	Context("fetching next orders", func() {
		It("returns empty response when FromPayload has no tradeable markets", func(ctx SpecContext) {
			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(BeEmpty())
			Expect(resp.HasMore).To(BeFalse())
		})

		It("converts URL symbol to slash format before calling the API", func(ctx SpecContext) {
			// "btcusd" must be converted to "BTC/USD" for the API call.
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(BeEmpty())
		})

		It("logs Info and skips market on 404 instead of erroring", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return(nil, &client.NotFoundError{Endpoint: "/api/v2/account_order_data/", Message: "not found"})
			m.EXPECT().GetAccountOrderData(gomock.Any(), "ETH/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd", "ethusd"}),
			})
			// 404 must not bubble up as an error; the cycle continues.
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(BeEmpty())
		})

		It("returns error and halts when a market fails with non-404", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return(nil, errors.New("boom"))

			_, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd", "ethusd"}),
			})
			Expect(err).To(HaveOccurred())
		})

		It("emits OPEN PSPOrder for order_created with no fills", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_created",
					EventID: "evt-1",
					Data: client.AccountOrderDataItem{
						ID: json.Number("1000"), IDStr: "1000",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779709892",
						AmountStr:      "0.00028416",
						AmountTraded:   "0",
						AmountAtCreate: "0.00028416",
						PriceStr:       "77400.00",
					},
				}}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			o := resp.Orders[0]
			Expect(o.Reference).To(Equal("1000"))
			Expect(o.Direction).To(Equal(models.ORDER_DIRECTION_BUY))
			Expect(o.Status).To(Equal(models.ORDER_STATUS_OPEN))
			Expect(o.Type).To(Equal(models.ORDER_TYPE_LIMIT))
			Expect(o.TimeInForce).To(Equal(models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED))
			Expect(o.BaseQuantityOrdered.Int64()).To(Equal(int64(28416)))
			Expect(o.BaseQuantityFilled.Sign()).To(Equal(0))
			Expect(o.LimitPrice.Int64()).To(Equal(int64(7740000)))
			Expect(o.SourceAsset).To(Equal("USD/2"))
			Expect(o.DestinationAsset).To(Equal("BTC/8"))
			Expect(resp.HasMore).To(BeFalse())
		})

		It("emits FILLED PSPOrder for order_deleted with full fill", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_deleted",
					EventID: "evt-2",
					Data: client.AccountOrderDataItem{
						ID: json.Number("1001"), IDStr: "1001",
						OrderType: 1, OrderSubtype: 0,
						Datetime:       "1779717950",
						AmountStr:      "0",
						AmountTraded:   "0.00028416",
						AmountAtCreate: "0.00028416",
						PriceStr:       "77400.00",
					},
				}}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			o := resp.Orders[0]
			Expect(o.Reference).To(Equal("1001"))
			Expect(o.Direction).To(Equal(models.ORDER_DIRECTION_SELL))
			Expect(o.Status).To(Equal(models.ORDER_STATUS_FILLED))
			Expect(o.BaseQuantityFilled.Int64()).To(Equal(int64(28416)))
		})

		It("emits CANCELLED PSPOrder for order_deleted with remaining amount", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_deleted",
					EventID: "evt-3",
					Data: client.AccountOrderDataItem{
						ID: json.Number("1002"), IDStr: "1002",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779717951",
						AmountStr:      "0.00028416",
						AmountTraded:   "0",
						AmountAtCreate: "0.00028416",
						PriceStr:       "77400.00",
					},
				}}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_CANCELLED))
		})

		It("handles scientific notation price (7.74E+4 = 77400)", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event: "order_created",
					Data: client.AccountOrderDataItem{
						ID: json.Number("1003"), IDStr: "1003",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779709892",
						AmountStr:      "0.00028416",
						AmountTraded:   "0",
						AmountAtCreate: "0.00028416",
						PriceStr:       "7.74E+4",
					},
				}}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			// 7.74E+4 = 77400.00 → 7740000 at quotePrec=2
			Expect(resp.Orders[0].LimitPrice.Int64()).To(Equal(int64(7740000)))
		})

		It("advances the since_id cursor (by URL-symbol key) and passes slash market on next call", func(ctx SpecContext) {
			const eventID = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_created",
					EventID: eventID,
					Data: client.AccountOrderDataItem{
						ID: json.Number("2000"), IDStr: "2000",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779709892",
						AmountStr:      "0.5",
						AmountTraded:   "0",
						AmountAtCreate: "0.5",
						PriceStr:       "60000.00",
					},
				}}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd"}),
			})
			Expect(err).ToNot(HaveOccurred())

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			// State key is URL symbol; value is the MarketEventID.
			Expect(state.LastSeenEventIDPerMarket["btcusd"]).To(Equal(eventID))

			// Second call: since_id=eventID passed to "BTC/USD" slash market.
			expectedSinceID := eventID
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", &expectedSinceID).
				Return([]client.AccountOrderDataEvent{}, nil)

			_, err = plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd"}),
				State:       resp.NewState,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("tracks the last EventID across multiple events in one batch", func(ctx SpecContext) {
			const firstEventID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			const lastEventID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{
					{
						Event:   "order_created",
						EventID: firstEventID,
						Data: client.AccountOrderDataItem{
							ID: json.Number("100"), IDStr: "100",
							OrderType: 0, OrderSubtype: 0,
							Datetime: "1779709892", AmountStr: "0.5",
							AmountTraded: "0", AmountAtCreate: "0.5", PriceStr: "60000.00",
						},
					},
					{
						Event:   "order_deleted",
						EventID: "cccccccccccccccccccccccccccccccc",
						Data: client.AccountOrderDataItem{
							ID: json.Number("100"), IDStr: "100",
							OrderType: 0, OrderSubtype: 0,
							Datetime: "1779717950", AmountStr: "0",
							AmountTraded: "0.5", AmountAtCreate: "0.5", PriceStr: "60000.00",
						},
					},
					{
						Event:   "order_created",
						EventID: lastEventID,
						Data: client.AccountOrderDataItem{
							ID: json.Number("200"), IDStr: "200",
							OrderType: 0, OrderSubtype: 0,
							Datetime: "1779717960", AmountStr: "1.0",
							AmountTraded: "0", AmountAtCreate: "1.0", PriceStr: "60000.00",
						},
					},
				}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets([]string{"btcusd"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(3))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			// Only the last EventID in the batch is kept as the cursor.
			Expect(state.LastSeenEventIDPerMarket["btcusd"]).To(Equal(lastEventID))
		})
	})
})
