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

	Context("fetching next orders", func() {
		It("returns the wrapped client error when open_orders fails", func(ctx SpecContext) {
			m.EXPECT().GetOpenOrders(gomock.Any()).Return(nil, errors.New("boom"))

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: []byte(`{}`)})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("boom"))
			Expect(err.Error()).To(ContainSubstring("fetch open orders"))
			Expect(resp).To(Equal(models.FetchNextOrdersResponse{}))
		})

		It("seeds tracked orders from a fresh snapshot and emits one OPEN PSPOrder per ID", func(ctx SpecContext) {
			m.EXPECT().GetOpenOrders(gomock.Any()).Return([]client.OpenOrder{
				{ID: "100", Datetime: "2025-09-25 14:00:00.000000", Type: "0", Price: "60000.00", Amount: "0.50000000", CurrencyPair: "btcusd"},
			}, nil)
			m.EXPECT().GetOrderStatus(gomock.Any(), "100").Return(client.OrderStatus{
				ID:              "100",
				Datetime:        "2025-09-25 14:00:00.000000",
				Type:            "0",
				Subtype:         mappers.OrderSubtypeLimit,
				Market:          "BTC/USD",
				AmountRemaining: "0.50000000",
				Status:          mappers.OrderStatusOpen,
			}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Reference).To(Equal("100"))
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_OPEN))
			Expect(resp.Orders[0].Direction).To(Equal(models.ORDER_DIRECTION_BUY))
			Expect(resp.Orders[0].BaseQuantityOrdered.Int64()).To(Equal(int64(50_000_000)))
			Expect(resp.HasMore).To(BeFalse())

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.TrackedOrders).To(HaveKey("100"))
			Expect(state.TrackedOrders["100"].LimitPrice).To(Equal("60000.00"))
		})

		It("marks a tracked order PARTIALLY_FILLED when order_status returns fills under Open", func(ctx SpecContext) {
			now := time.Now().UTC()
			initial := ordersState{TrackedOrders: map[string]trackedOrder{
				"101": {LastStatus: mappers.OrderStatusOpen, FirstSeenAt: now.Add(-time.Hour),
					LimitPrice: "60000.00"},
			}}
			rawState, _ := json.Marshal(initial)

			m.EXPECT().GetOpenOrders(gomock.Any()).Return([]client.OpenOrder{
				{ID: "101", Type: "1", Price: "60000.00", Amount: "1.00000000", CurrencyPair: "btcusd"},
			}, nil)
			m.EXPECT().GetOrderStatus(gomock.Any(), "101").Return(client.OrderStatus{
				ID:              "101",
				Type:            "1",
				Subtype:         mappers.OrderSubtypeLimit,
				Market:          "BTC/USD",
				AmountRemaining: "0.75000000",
				Status:          mappers.OrderStatusOpen,
				Transactions: []client.OrderTransaction{{
					TID:             1,
					Price:           "60000.00",
					Fee:             "7.50",
					CurrencyAmounts: map[string]string{"btc": "0.25", "usd": "15000.00"},
				}},
			}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: rawState})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_PARTIALLY_FILLED))
			Expect(resp.Orders[0].BaseQuantityFilled.Int64()).To(Equal(int64(25_000_000)))
			Expect(resp.Orders[0].QuoteAmount.Int64()).To(Equal(int64(1_500_000)))
			Expect(resp.Orders[0].Fee.Int64()).To(Equal(int64(750)))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.TrackedOrders).To(HaveKey("101"))
		})

		It("drops a tracked order from state when it becomes FILLED", func(ctx SpecContext) {
			initial := ordersState{TrackedOrders: map[string]trackedOrder{
				"102": {LastStatus: mappers.OrderStatusOpen, FirstSeenAt: time.Now().Add(-time.Hour),
					LimitPrice: "60000.00"},
			}}
			rawState, _ := json.Marshal(initial)

			m.EXPECT().GetOpenOrders(gomock.Any()).Return([]client.OpenOrder{}, nil)
			m.EXPECT().GetOrderStatus(gomock.Any(), "102").Return(client.OrderStatus{
				ID:              "102",
				Type:            "0",
				Subtype:         mappers.OrderSubtypeLimit,
				Market:          "BTC/USD",
				AmountRemaining: "0",
				Status:          mappers.OrderStatusFinished,
				Transactions: []client.OrderTransaction{{
					TID: 1, Price: "60000.00", Fee: "15.00",
					CurrencyAmounts: map[string]string{"btc": "0.5", "usd": "30000.00"},
				}},
			}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: rawState})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_FILLED))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.TrackedOrders).ToNot(HaveKey("102"))
		})

		It("drops a tracked order from state when it becomes CANCELLED", func(ctx SpecContext) {
			initial := ordersState{TrackedOrders: map[string]trackedOrder{
				"103": {LastStatus: mappers.OrderStatusOpen, FirstSeenAt: time.Now().Add(-time.Hour),
					LimitPrice: "60000.00"},
			}}
			rawState, _ := json.Marshal(initial)

			m.EXPECT().GetOpenOrders(gomock.Any()).Return([]client.OpenOrder{}, nil)
			m.EXPECT().GetOrderStatus(gomock.Any(), "103").Return(client.OrderStatus{
				ID:      "103",
				Type:    "0",
				Subtype: mappers.OrderSubtypeLimit,
				Market:  "BTC/USD",
				Status:  mappers.OrderStatusCanceled,
			}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: rawState})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_CANCELLED))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.TrackedOrders).ToNot(HaveKey("103"))
		})

		It("evicts long-lived tracked orders past orderRetentionMax with metadata flag", func(ctx SpecContext) {
			old := time.Now().UTC().Add(-orderRetentionMax - time.Minute)
			initial := ordersState{TrackedOrders: map[string]trackedOrder{
				"200": {LastStatus: mappers.OrderStatusOpen, FirstSeenAt: old,
					LimitPrice: "60000.00"},
			}}
			rawState, _ := json.Marshal(initial)

			m.EXPECT().GetOpenOrders(gomock.Any()).Return([]client.OpenOrder{
				{ID: "200", Type: "0", Price: "60000.00", Amount: "0.50000000", CurrencyPair: "btcusd"},
			}, nil)
			m.EXPECT().GetOrderStatus(gomock.Any(), "200").Return(client.OrderStatus{
				ID:      "200",
				Type:    "0",
				Subtype: mappers.OrderSubtypeLimit,
				Market:  "BTC/USD",
				Status:  mappers.OrderStatusOpen,
			}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: rawState})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Metadata[mappers.MetadataKeyRetentionExpired]).To(Equal("true"))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.TrackedOrders).ToNot(HaveKey("200"))
		})

		It("does not poison the cycle when one order_status call fails", func(ctx SpecContext) {
			m.EXPECT().GetOpenOrders(gomock.Any()).Return([]client.OpenOrder{
				{ID: "300", Type: "0", Price: "1.00", Amount: "1.00000000", CurrencyPair: "btcusd"},
				{ID: "301", Type: "0", Price: "1.00", Amount: "1.00000000", CurrencyPair: "btcusd"},
			}, nil)
			m.EXPECT().GetOrderStatus(gomock.Any(), "300").Return(client.OrderStatus{}, errors.New("transient"))
			m.EXPECT().GetOrderStatus(gomock.Any(), "301").Return(client.OrderStatus{
				ID:      "301",
				Type:    "0",
				Subtype: mappers.OrderSubtypeLimit,
				Market:  "BTC/USD",
				Status:  mappers.OrderStatusOpen,
			}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Reference).To(Equal("301"))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.TrackedOrders).To(HaveKey("300"))
			Expect(state.TrackedOrders).To(HaveKey("301"))
		})

		It("handles MARKET subtype where the order never appeared in open_orders/", func(ctx SpecContext) {
			// MARKET orders that fully fill within one cycle bypass
			// open_orders/. The orchestrator must still emit a sensible
			// PSPOrder when the order_status reply lands (e.g. via the
			// trailing-poll path of a previously-tracked order). Here
			// we drive that with a state that already tracks the ID
			// (LimitPrice empty per MAPPINGS §3.4.1).
			initial := ordersState{TrackedOrders: map[string]trackedOrder{
				"400": {LastStatus: mappers.OrderStatusOpen, FirstSeenAt: time.Now().UTC()},
			}}
			rawState, _ := json.Marshal(initial)

			m.EXPECT().GetOpenOrders(gomock.Any()).Return([]client.OpenOrder{}, nil)
			m.EXPECT().GetOrderStatus(gomock.Any(), "400").Return(client.OrderStatus{
				ID:      "400",
				Type:    "0",
				Subtype: mappers.OrderSubtypeMarket,
				Market:  "BTC/USD",
				Status:  mappers.OrderStatusFinished,
				Transactions: []client.OrderTransaction{{
					TID: 5, Price: "60000.00", Fee: "15.00",
					CurrencyAmounts: map[string]string{"btc": "0.5", "usd": "30000.00"},
				}},
			}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: rawState})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Type).To(Equal(models.ORDER_TYPE_MARKET))
			Expect(resp.Orders[0].TimeInForce).To(Equal(models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL))
			Expect(resp.Orders[0].LimitPrice).To(BeNil(), "MARKET order should not have a LimitPrice")
		})
	})
})
