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
	// Markets must be in slash format (e.g. "BTC/USD"); reference is the
	// currency that owns the orders being fetched.
	fromPayloadWithMarkets := func(reference string, markets []string) json.RawMessage {
		marketsRaw, _ := json.Marshal(markets)
		account := models.PSPAccount{
			Reference: reference,
			Metadata:  map[string]string{mappers.MetadataKeyTradableMarkets: string(marketsRaw)},
		}
		payload, _ := json.Marshal(account)
		return payload
	}

	Context("fetching next orders", func() {
		It("returns error when FromPayload is empty", func(ctx SpecContext) {
			_, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing payload in FromPayload"))
		})

		It("passes the market name directly to the API", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
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
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD", "ETH/USD"}),
			})
			// 404 must not bubble up as an error; the cycle continues.
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(BeEmpty())
		})

		It("returns error and halts when a market fails with non-404", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return(nil, errors.New("boom"))

			_, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD", "ETH/USD"}),
			})
			Expect(err).To(HaveOccurred())
		})

		It("emits OPEN PSPOrder for order_created with no fills", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_created",
					EventID: "00000000-0000-0000-0000-000000000001",
					Data: client.AccountOrderDataItem{
						ID: json.Number("1000"), IDStr: "1000",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779709892",
						Amount:         json.Number("0.00028416"),
						AmountStr:      "0.00028416",
						AmountTraded:   "0",
						AmountAtCreate: "0.00028416",
						PriceStr:       "77400.00",
					},
				}}, nil)

			// BUY order: source is quote (USD), so reference must be "USD".
			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
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
			Expect(o.SourceAccountReference).ToNot(BeNil())
			Expect(*o.SourceAccountReference).To(Equal("USD"))
			Expect(o.DestinationAccountReference).To(BeNil())
			Expect(resp.HasMore).To(BeFalse())
		})

		It("emits FILLED PSPOrder for order_deleted with full fill", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_deleted",
					EventID: "00000000-0000-0000-0000-000000000002",
					Data: client.AccountOrderDataItem{
						ID: json.Number("1001"), IDStr: "1001",
						OrderType: 1, OrderSubtype: 0,
						Datetime:       "1779717950",
						Amount:         json.Number("0.00028416"),
						AmountStr:      "0",
						AmountTraded:   "0.00028416",
						AmountAtCreate: "0.00028416",
						PriceStr:       "77400.00",
					},
				}}, nil)

			// SELL order: source is base (BTC), so reference must be "BTC".
			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("BTC", []string{"BTC/USD"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			o := resp.Orders[0]
			Expect(o.Reference).To(Equal("1001"))
			Expect(o.Direction).To(Equal(models.ORDER_DIRECTION_SELL))
			Expect(o.Status).To(Equal(models.ORDER_STATUS_FILLED))
			Expect(o.BaseQuantityFilled.Int64()).To(Equal(int64(28416)))
			Expect(o.SourceAccountReference).ToNot(BeNil())
			Expect(*o.SourceAccountReference).To(Equal("BTC"))
			Expect(o.DestinationAccountReference).To(BeNil())
		})

		It("emits CANCELLED PSPOrder for order_deleted with remaining amount", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_deleted",
					EventID: "00000000-0000-0000-0000-000000000003",
					Data: client.AccountOrderDataItem{
						ID: json.Number("1002"), IDStr: "1002",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779717951",
						Amount:         json.Number("0.00028416"),
						AmountStr:      "0.00028416",
						AmountTraded:   "0",
						AmountAtCreate: "0.00028416",
						PriceStr:       "77400.00",
					},
				}}, nil)

			// BUY order: source is quote (USD), so reference must be "USD".
			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_CANCELLED))
		})

		It("handles scientific notation price (7.74E+4 = 77400)", func(ctx SpecContext) {
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_created",
					EventID: "00000000-0000-0000-0000-000000000004",
					Data: client.AccountOrderDataItem{
						ID: json.Number("1003"), IDStr: "1003",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779709892",
						Amount:         json.Number("0.00028416"),
						AmountStr:      "0.00028416",
						AmountTraded:   "0",
						AmountAtCreate: "0.00028416",
						PriceStr:       "7.74E+4",
					},
				}}, nil)

			// BUY order: source is quote (USD), so reference must be "USD".
			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			// 7.74E+4 = 77400.00 → 7740000 at quotePrec=2
			Expect(resp.Orders[0].LimitPrice.Int64()).To(Equal(int64(7740000)))
		})

		It("advances the since_id cursor and passes it on the next call", func(ctx SpecContext) {
			const eventID = "a1b2c3d4-e5f6-a1b2-c3d4-e5f6a1b2c3d4"
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_created",
					EventID: eventID,
					Data: client.AccountOrderDataItem{
						ID: json.Number("2000"), IDStr: "2000",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779709892",
						Amount:         json.Number("0.5"),
						AmountStr:      "0.5",
						AmountTraded:   "0",
						AmountAtCreate: "0.5",
						PriceStr:       "60000.00",
					},
				}}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
			})
			Expect(err).ToNot(HaveOccurred())

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			// State key is the market name as passed in FromPayload.
			Expect(state.LastSeenEventIDPerMarket["BTC/USD"]).To(Equal(eventID))

			// Second call: since_id=eventID passed for the same market.
			expectedSinceID := eventID
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", &expectedSinceID).
				Return([]client.AccountOrderDataEvent{}, nil)

			_, err = plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
				State:       resp.NewState,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("skips the event whose ID equals the since_id cursor to avoid duplicate imports", func(ctx SpecContext) {
			const sinceID = "a1b2c3d4-e5f6-a1b2-c3d4-e5f6a1b2c3d4"
			const newEventID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

			initialState, _ := json.Marshal(ordersState{
				LastSeenEventIDPerMarket: map[string]string{"BTC/USD": sinceID},
			})

			// API returns the since_id event first (as Bitstamp does), followed by a new event.
			expectedSinceID := sinceID
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", &expectedSinceID).
				Return([]client.AccountOrderDataEvent{
					{
						Event:   "order_created",
						EventID: sinceID,
						Data: client.AccountOrderDataItem{
							ID: json.Number("1000"), IDStr: "1000",
							OrderType: 0, OrderSubtype: 0,
							Datetime:       "1779709892",
							Amount:         json.Number("0.5"),
							AmountStr:      "0.5",
							AmountTraded:   "0",
							AmountAtCreate: "0.5",
							PriceStr:       "60000.00",
						},
					},
					{
						Event:   "order_created",
						EventID: newEventID,
						Data: client.AccountOrderDataItem{
							ID: json.Number("1001"), IDStr: "1001",
							OrderType: 0, OrderSubtype: 0,
							Datetime:       "1779709900",
							Amount:         json.Number("0.5"),
							AmountStr:      "0.5",
							AmountTraded:   "0",
							AmountAtCreate: "0.5",
							PriceStr:       "60000.00",
						},
					},
				}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
				State:       initialState,
			})
			Expect(err).ToNot(HaveOccurred())
			// The since_id event must be dropped; only the new event is returned.
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Reference).To(Equal("1001"))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.LastSeenEventIDPerMarket["BTC/USD"]).To(Equal(newEventID))
		})

		It("truncates orders within a market when the API returns more events than the page size", func(ctx SpecContext) {
			const firstEventID = "a1b2c3d4-e5f6-a1b2-c3d4-e5f6a1b2c3d4"
			const secondEventID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{
					{
						Event:   "order_created",
						EventID: firstEventID,
						Data: client.AccountOrderDataItem{
							ID: json.Number("5000"), IDStr: "5000",
							OrderType: 0, OrderSubtype: 0,
							Datetime:       "1779709892",
							Amount:         json.Number("0.5"),
							AmountStr:      "0.5",
							AmountTraded:   "0",
							AmountAtCreate: "0.5",
							PriceStr:       "60000.00",
						},
					},
					{
						Event:   "order_created",
						EventID: secondEventID,
						Data: client.AccountOrderDataItem{
							ID: json.Number("5001"), IDStr: "5001",
							OrderType: 0, OrderSubtype: 0,
							Datetime:       "1779709893",
							Amount:         json.Number("0.5"),
							AmountStr:      "0.5",
							AmountTraded:   "0",
							AmountAtCreate: "0.5",
							PriceStr:       "60000.00",
						},
					},
				}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				PageSize:    1,
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Reference).To(Equal("5000"))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.HasMoreCurrentMarket).To(Equal("BTC/USD"))
			// Cursor must point to the last event included in this page, not the one we didn't return.
			Expect(state.LastSeenEventIDPerMarket["BTC/USD"]).To(Equal(firstEventID))
		})

		It("returns HasMore=true and saves HasMoreCurrentMarket when page limit is reached", func(ctx SpecContext) {
			const eventID = "a1b2c3d4-e5f6-a1b2-c3d4-e5f6a1b2c3d4"
			// BTC/USD returns one order (fills the page), ETH/USD must not be called.
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_created",
					EventID: eventID,
					Data: client.AccountOrderDataItem{
						ID: json.Number("3000"), IDStr: "3000",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779709892",
						Amount:         json.Number("0.5"),
						AmountStr:      "0.5",
						AmountTraded:   "0",
						AmountAtCreate: "0.5",
						PriceStr:       "60000.00",
					},
				}}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				PageSize:    1,
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD", "ETH/USD"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.Orders).To(HaveLen(1))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.HasMoreCurrentMarket).To(Equal("BTC/USD"))
			Expect(state.LastSeenEventIDPerMarket["BTC/USD"]).To(Equal(eventID))
		})

		It("resumes from HasMoreCurrentMarket and skips earlier markets", func(ctx SpecContext) {
			const btcEventID = "a1b2c3d4-e5f6-a1b2-c3d4-e5f6a1b2c3d4"
			const ethEventID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

			// Simulate state saved after first page stopped at BTC/USD.
			initialState, _ := json.Marshal(ordersState{
				LastSeenEventIDPerMarket: map[string]string{"BTC/USD": btcEventID},
				HasMoreCurrentMarket:     "BTC/USD",
			})

			// BTC/USD is re-fetched from its since_id; ETH/USD is now called.
			expectedSinceID := btcEventID
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", &expectedSinceID).
				Return([]client.AccountOrderDataEvent{}, nil)
			m.EXPECT().GetAccountOrderData(gomock.Any(), "ETH/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{{
					Event:   "order_created",
					EventID: ethEventID,
					Data: client.AccountOrderDataItem{
						ID: json.Number("4000"), IDStr: "4000",
						OrderType: 0, OrderSubtype: 0,
						Datetime:       "1779709892",
						Amount:         json.Number("1.0"),
						AmountStr:      "1.0",
						AmountTraded:   "0",
						AmountAtCreate: "1.0",
						PriceStr:       "2000.00",
					},
				}}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				PageSize:    10,
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD", "ETH/USD"}),
				State:       initialState,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.Orders).To(HaveLen(1))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.HasMoreCurrentMarket).To(BeEmpty())
			Expect(state.LastSeenEventIDPerMarket["ETH/USD"]).To(Equal(ethEventID))
		})

		It("clears HasMoreCurrentMarket when all markets fit within page size", func(ctx SpecContext) {
			initialState, _ := json.Marshal(ordersState{
				LastSeenEventIDPerMarket: map[string]string{},
				HasMoreCurrentMarket:     "BTC/USD",
			})
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				PageSize:    10,
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
				State:       initialState,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HasMore).To(BeFalse())

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.HasMoreCurrentMarket).To(BeEmpty())
		})

		It("tracks the last EventID across multiple events in one batch", func(ctx SpecContext) {
			const firstEventID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
			const lastEventID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
			m.EXPECT().GetAccountOrderData(gomock.Any(), "BTC/USD", gomock.Nil()).
				Return([]client.AccountOrderDataEvent{
					{
						Event:   "order_created",
						EventID: firstEventID,
						Data: client.AccountOrderDataItem{
							ID: json.Number("100"), IDStr: "100",
							OrderType: 0, OrderSubtype: 0,
							Datetime: "1779709892", Amount: json.Number("0.5"), AmountStr: "0.5",
							AmountTraded: "0", AmountAtCreate: "0.5", PriceStr: "60000.00",
						},
					},
					{
						Event:   "order_deleted",
						EventID: "cccccccc-cccc-cccc-cccc-cccccccccccc",
						Data: client.AccountOrderDataItem{
							ID: json.Number("100"), IDStr: "100",
							OrderType: 0, OrderSubtype: 0,
							Datetime: "1779717950", Amount: json.Number("0.5"), AmountStr: "0",
							AmountTraded: "0.5", AmountAtCreate: "0.5", PriceStr: "60000.00",
						},
					},
					{
						Event:   "order_created",
						EventID: lastEventID,
						Data: client.AccountOrderDataItem{
							ID: json.Number("200"), IDStr: "200",
							OrderType: 0, OrderSubtype: 0,
							Datetime: "1779717960", Amount: json.Number("1.0"), AmountStr: "1.0",
							AmountTraded: "0", AmountAtCreate: "1.0", PriceStr: "60000.00",
						},
					},
				}, nil)

			// All three are BUY orders (orderType=0), so reference must be "USD".
			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{
				FromPayload: fromPayloadWithMarkets("USD", []string{"BTC/USD"}),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(3))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			// Only the last EventID in the batch is kept as the cursor.
			Expect(state.LastSeenEventIDPerMarket["BTC/USD"]).To(Equal(lastEventID))
		})
	})
})
