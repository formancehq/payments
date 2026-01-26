package kraken

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Kraken Plugin Orders", func() {
	var (
		ctrl   *gomock.Controller
		m      *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			client: m,
			logger: logger,
			config: Config{
				Endpoint: "https://api.kraken.com",
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetch next orders", func() {
		It("fetches open and closed orders successfully", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			openOrders := &client.OpenOrdersResponse{
				Orders: map[string]client.Order{
					"OPEN-ORDER-1": {
						Status:   "open",
						OpenTime: 1700000000,
						Vol:      "1.0",
						VolExec:  "0",
						Fee:      "0",
						Descr: client.OrderDesc{
							Pair:      "XBTUSD",
							Type:      "buy",
							OrderType: "limit",
							Price:     "50000.00",
						},
					},
				},
			}

			closedOrders := &client.ClosedOrdersResponse{
				Orders: map[string]client.Order{
					"CLOSED-ORDER-1": {
						Status:    "closed",
						OpenTime:  1699900000,
						CloseTime: 1699900100,
						Vol:       "0.5",
						VolExec:   "0.5",
						Fee:       "0.001",
						Descr: client.OrderDesc{
							Pair:      "ETHUSD",
							Type:      "sell",
							OrderType: "market",
						},
					},
				},
				Count: 1,
			}

			m.EXPECT().GetOpenOrders(gomock.Any(), gomock.Any()).Return(openOrders, nil)
			m.EXPECT().GetClosedOrders(gomock.Any(), gomock.Any()).Return(closedOrders, nil)

			res, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Orders).To(HaveLen(2))

			// Find the orders
			var openOrder, closedOrder models.PSPOrder
			for _, o := range res.Orders {
				if o.Reference == "OPEN-ORDER-1" {
					openOrder = o
				} else if o.Reference == "CLOSED-ORDER-1" {
					closedOrder = o
				}
			}

			// Verify open order
			Expect(openOrder.Status).To(Equal(models.ORDER_STATUS_OPEN))
			Expect(openOrder.Direction).To(Equal(models.ORDER_DIRECTION_BUY))
			Expect(openOrder.Type).To(Equal(models.ORDER_TYPE_LIMIT))

			// Verify closed order
			Expect(closedOrder.Status).To(Equal(models.ORDER_STATUS_FILLED))
			Expect(closedOrder.Direction).To(Equal(models.ORDER_DIRECTION_SELL))
			Expect(closedOrder.Type).To(Equal(models.ORDER_TYPE_MARKET))
		})

		It("returns error on open orders client failure", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetOpenOrders(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("client error"))

			_, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get open orders"))
		})

		It("returns error on closed orders client failure", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetOpenOrders(gomock.Any(), gomock.Any()).Return(&client.OpenOrdersResponse{Orders: map[string]client.Order{}}, nil)
			m.EXPECT().GetClosedOrders(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("client error"))

			_, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get closed orders"))
		})

		It("returns error on invalid state", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    json.RawMessage(`{invalid json}`),
				PageSize: 100,
			}

			_, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal state"))
		})
	})

	Context("create order", func() {
		It("creates a market buy order successfully", func(ctx SpecContext) {
			req := models.CreateOrderRequest{
				Order: models.PSPOrder{
					Reference:           "client-order-1",
					Direction:           models.ORDER_DIRECTION_BUY,
					Type:                models.ORDER_TYPE_MARKET,
					SourceAsset:         "BTC",
					TargetAsset:         "USD",
					BaseQuantityOrdered: big.NewInt(100000000),
					TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					Description: "buy 1 BTC @ market",
					TxID:        []string{"ORDER-TX-ID"},
				},
				nil,
			)

			res, err := plg.CreateOrder(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.PollingOrderID).NotTo(BeNil())
			Expect(*res.PollingOrderID).To(Equal("ORDER-TX-ID"))
		})

		It("creates a limit sell order successfully", func(ctx SpecContext) {
			limitPrice := big.NewInt(5000000)
			req := models.CreateOrderRequest{
				Order: models.PSPOrder{
					Reference:           "client-order-2",
					Direction:           models.ORDER_DIRECTION_SELL,
					Type:                models.ORDER_TYPE_LIMIT,
					SourceAsset:         "BTC",
					TargetAsset:         "USD",
					BaseQuantityOrdered: big.NewInt(50000000),
					LimitPrice:          limitPrice,
					TimeInForce:         models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL,
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					Description: "sell 0.5 BTC @ 50000.00",
					TxID:        []string{"LIMIT-ORDER-TX-ID"},
				},
				nil,
			)

			res, err := plg.CreateOrder(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.PollingOrderID).NotTo(BeNil())
			Expect(*res.PollingOrderID).To(Equal("LIMIT-ORDER-TX-ID"))
		})

		It("returns error on create failure", func(ctx SpecContext) {
			req := models.CreateOrderRequest{
				Order: models.PSPOrder{
					Reference:           "client-order-3",
					Direction:           models.ORDER_DIRECTION_BUY,
					Type:                models.ORDER_TYPE_MARKET,
					SourceAsset:         "BTC",
					TargetAsset:         "USD",
					BaseQuantityOrdered: big.NewInt(100000000),
					TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("create order failed"))

			_, err := plg.CreateOrder(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create order"))
		})
	})

	Context("cancel order", func() {
		It("cancels an order successfully", func(ctx SpecContext) {
			req := models.CancelOrderRequest{
				OrderID: "ORDER-TO-CANCEL",
			}

			m.EXPECT().CancelOrder(gomock.Any(), "ORDER-TO-CANCEL").Return(
				&client.CancelOrderResponse{
					Count: 1,
				},
				nil,
			)

			res, err := plg.CancelOrder(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Order.Reference).To(Equal("ORDER-TO-CANCEL"))
			Expect(res.Order.Status).To(Equal(models.ORDER_STATUS_CANCELLED))
		})

		It("returns error on cancel failure", func(ctx SpecContext) {
			req := models.CancelOrderRequest{
				OrderID: "ORDER-TO-CANCEL",
			}

			m.EXPECT().CancelOrder(gomock.Any(), "ORDER-TO-CANCEL").Return(nil, fmt.Errorf("cancel failed"))

			_, err := plg.CancelOrder(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to cancel order"))
		})
	})

	Context("status mapping", func() {
		It("maps pending status correctly", func() {
			status := mapKrakenStatus("pending", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_PENDING))
		})

		It("maps open status correctly", func() {
			status := mapKrakenStatus("open", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_OPEN))
		})

		It("maps open with partial fill to PARTIALLY_FILLED", func() {
			status := mapKrakenStatus("open", "1.0", "0.5")
			Expect(status).To(Equal(models.ORDER_STATUS_PARTIALLY_FILLED))
		})

		It("maps closed status to FILLED", func() {
			status := mapKrakenStatus("closed", "1.0", "1.0")
			Expect(status).To(Equal(models.ORDER_STATUS_FILLED))
		})

		It("maps canceled status correctly", func() {
			status := mapKrakenStatus("canceled", "1.0", "0.5")
			Expect(status).To(Equal(models.ORDER_STATUS_CANCELLED))
		})

		It("maps cancelled status correctly (British spelling)", func() {
			status := mapKrakenStatus("cancelled", "1.0", "0.5")
			Expect(status).To(Equal(models.ORDER_STATUS_CANCELLED))
		})

		It("maps expired status correctly", func() {
			status := mapKrakenStatus("expired", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_EXPIRED))
		})

		It("maps unknown status to PENDING", func() {
			status := mapKrakenStatus("unknown", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_PENDING))
		})
	})

	Context("pair building", func() {
		It("builds Kraken pairs correctly", func() {
			pair := buildKrakenPair("BTC", "USD")
			Expect(pair).To(Equal("BTCUSD"))

			pair = buildKrakenPair("ETH", "EUR")
			Expect(pair).To(Equal("ETHEUR"))
		})
	})
})
