package coinbaseprime

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/coinbase-samples/prime-sdk-go/model"
	"github.com/coinbase-samples/prime-sdk-go/orders"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Prime Plugin Orders", func() {
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
				PortfolioID: "test-portfolio-id",
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetch next orders", func() {
		var sampleOrders []*model.Order

		BeforeEach(func() {
			now := time.Now()
			sampleOrders = []*model.Order{
				{
					Id:             "order-1",
					ProductId:      "BTC-USD",
					Side:           "BUY",
					Type:           "MARKET",
					BaseQuantity:   "1.0",
					FilledQuantity: "1.0",
					Status:         "FILLED",
					TimeInForce:    "GTC",
					Created:        now.Add(-1 * time.Hour).Format(time.RFC3339),
				},
				{
					Id:             "order-2",
					ProductId:      "ETH-USD",
					Side:           "SELL",
					Type:           "LIMIT",
					BaseQuantity:   "5.0",
					FilledQuantity: "2.5",
					LimitPrice:     "2000.00",
					Status:         "OPEN",
					TimeInForce:    "IOC",
					Created:        now.Format(time.RFC3339),
				},
			}
		})

		It("fetches orders successfully", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().ListOrders(gomock.Any(), gomock.Any()).Return(
				&orders.ListOrdersResponse{
					Orders: sampleOrders,
					Pagination: &model.Pagination{
						NextCursor: "next-cursor",
						HasNext:    true,
					},
				},
				nil,
			)

			res, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.Orders).To(HaveLen(2))
			Expect(res.Orders[0].Reference).To(Equal("order-1"))
			Expect(res.Orders[0].Direction).To(Equal(models.ORDER_DIRECTION_BUY))
			Expect(res.Orders[0].Type).To(Equal(models.ORDER_TYPE_MARKET))
			Expect(res.Orders[0].Status).To(Equal(models.ORDER_STATUS_FILLED))
			Expect(res.Orders[1].Reference).To(Equal("order-2"))
			Expect(res.Orders[1].Direction).To(Equal(models.ORDER_DIRECTION_SELL))
			Expect(res.Orders[1].Type).To(Equal(models.ORDER_TYPE_LIMIT))
			Expect(res.Orders[1].Status).To(Equal(models.ORDER_STATUS_PARTIALLY_FILLED))

			var state ordersState
			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Cursor).To(Equal("next-cursor"))
		})

		It("fetches orders with cursor", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    json.RawMessage(`{"cursor": "previous-cursor"}`),
				PageSize: 100,
			}

			m.EXPECT().ListOrders(gomock.Any(), gomock.Any()).Return(
				&orders.ListOrdersResponse{
					Orders: sampleOrders[:1],
					Pagination: &model.Pagination{
						NextCursor: "",
						HasNext:    false,
					},
				},
				nil,
			)

			res, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Orders).To(HaveLen(1))
		})

		It("returns error on client failure", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			m.EXPECT().ListOrders(gomock.Any(), gomock.Any()).Return(
				nil,
				fmt.Errorf("client error"),
			)

			_, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to list orders"))
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
					BaseQuantityOrdered: big.NewInt(100000000), // 1 BTC
					TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&orders.CreateOrderResponse{
					OrderId: "created-order-id",
				},
				nil,
			)

			m.EXPECT().GetOrder(gomock.Any(), "created-order-id").Return(
				&orders.GetOrderResponse{
					Order: &model.Order{
						Id:             "created-order-id",
						ProductId:      "BTC-USD",
						Side:           "BUY",
						Type:           "MARKET",
						BaseQuantity:   "1.0",
						FilledQuantity: "0",
						Status:         "PENDING",
						TimeInForce:    "GTC",
						Created:        time.Now().Format(time.RFC3339),
					},
				},
				nil,
			)

			res, err := plg.CreateOrder(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Order).NotTo(BeNil())
			Expect(res.Order.Reference).To(Equal("created-order-id"))
		})

		It("creates a limit sell order successfully", func(ctx SpecContext) {
			limitPrice := big.NewInt(5000000) // $50000
			req := models.CreateOrderRequest{
				Order: models.PSPOrder{
					Reference:           "client-order-2",
					Direction:           models.ORDER_DIRECTION_SELL,
					Type:                models.ORDER_TYPE_LIMIT,
					SourceAsset:         "BTC",
					TargetAsset:         "USD",
					BaseQuantityOrdered: big.NewInt(50000000), // 0.5 BTC
					LimitPrice:          limitPrice,
					TimeInForce:         models.TIME_IN_FORCE_FILL_OR_KILL,
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&orders.CreateOrderResponse{
					OrderId: "created-limit-order-id",
				},
				nil,
			)

			m.EXPECT().GetOrder(gomock.Any(), "created-limit-order-id").Return(
				&orders.GetOrderResponse{
					Order: &model.Order{
						Id:             "created-limit-order-id",
						ProductId:      "BTC-USD",
						Side:           "SELL",
						Type:           "LIMIT",
						BaseQuantity:   "0.5",
						FilledQuantity: "0",
						LimitPrice:     "50000.00",
						Status:         "OPEN",
						TimeInForce:    "FOK",
						Created:        time.Now().Format(time.RFC3339),
					},
				},
				nil,
			)

			res, err := plg.CreateOrder(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Order).NotTo(BeNil())
			Expect(res.Order.Reference).To(Equal("created-limit-order-id"))
			Expect(res.Order.Type).To(Equal(models.ORDER_TYPE_LIMIT))
		})

		It("returns polling ID when get order fails", func(ctx SpecContext) {
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

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&orders.CreateOrderResponse{
					OrderId: "created-order-id",
				},
				nil,
			)

			m.EXPECT().GetOrder(gomock.Any(), "created-order-id").Return(
				nil,
				fmt.Errorf("get order failed"),
			)

			res, err := plg.CreateOrder(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Order).To(BeNil())
			Expect(res.PollingOrderID).NotTo(BeNil())
			Expect(*res.PollingOrderID).To(Equal("created-order-id"))
		})

		It("returns error on create failure", func(ctx SpecContext) {
			req := models.CreateOrderRequest{
				Order: models.PSPOrder{
					Reference:           "client-order-4",
					Direction:           models.ORDER_DIRECTION_BUY,
					Type:                models.ORDER_TYPE_MARKET,
					SourceAsset:         "BTC",
					TargetAsset:         "USD",
					BaseQuantityOrdered: big.NewInt(100000000),
					TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				nil,
				fmt.Errorf("create order failed"),
			)

			_, err := plg.CreateOrder(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create order"))
		})
	})

	Context("cancel order", func() {
		It("cancels an order successfully", func(ctx SpecContext) {
			req := models.CancelOrderRequest{
				OrderID: "order-to-cancel",
			}

			m.EXPECT().CancelOrder(gomock.Any(), "order-to-cancel").Return(
				&orders.CancelOrderResponse{},
				nil,
			)

			m.EXPECT().GetOrder(gomock.Any(), "order-to-cancel").Return(
				&orders.GetOrderResponse{
					Order: &model.Order{
						Id:             "order-to-cancel",
						ProductId:      "BTC-USD",
						Side:           "BUY",
						Type:           "MARKET",
						BaseQuantity:   "1.0",
						FilledQuantity: "0.5",
						Status:         "CANCELLED",
						TimeInForce:    "GTC",
						Created:        time.Now().Format(time.RFC3339),
					},
				},
				nil,
			)

			res, err := plg.CancelOrder(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Order.Reference).To(Equal("order-to-cancel"))
			Expect(res.Order.Status).To(Equal(models.ORDER_STATUS_CANCELLED))
		})

		It("returns error on cancel failure", func(ctx SpecContext) {
			req := models.CancelOrderRequest{
				OrderID: "order-to-cancel",
			}

			m.EXPECT().CancelOrder(gomock.Any(), "order-to-cancel").Return(
				nil,
				fmt.Errorf("cancel failed"),
			)

			_, err := plg.CancelOrder(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to cancel order"))
		})

		It("returns error when get cancelled order fails", func(ctx SpecContext) {
			req := models.CancelOrderRequest{
				OrderID: "order-to-cancel",
			}

			m.EXPECT().CancelOrder(gomock.Any(), "order-to-cancel").Return(
				&orders.CancelOrderResponse{},
				nil,
			)

			m.EXPECT().GetOrder(gomock.Any(), "order-to-cancel").Return(
				nil,
				fmt.Errorf("get order failed"),
			)

			_, err := plg.CancelOrder(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get cancelled order"))
		})
	})

	Context("status mapping", func() {
		It("maps PENDING status correctly", func() {
			status := mapCoinbaseStatus("PENDING", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_PENDING))
		})

		It("maps OPEN status correctly", func() {
			status := mapCoinbaseStatus("OPEN", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_OPEN))
		})

		It("maps OPEN with partial fill to PARTIALLY_FILLED", func() {
			status := mapCoinbaseStatus("OPEN", "1.0", "0.5")
			Expect(status).To(Equal(models.ORDER_STATUS_PARTIALLY_FILLED))
		})

		It("maps FILLED status correctly", func() {
			status := mapCoinbaseStatus("FILLED", "1.0", "1.0")
			Expect(status).To(Equal(models.ORDER_STATUS_FILLED))
		})

		It("maps CANCELLED status correctly", func() {
			status := mapCoinbaseStatus("CANCELLED", "1.0", "0.5")
			Expect(status).To(Equal(models.ORDER_STATUS_CANCELLED))
		})

		It("maps EXPIRED status correctly", func() {
			status := mapCoinbaseStatus("EXPIRED", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_EXPIRED))
		})

		It("maps FAILED status correctly", func() {
			status := mapCoinbaseStatus("FAILED", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_FAILED))
		})

		It("maps unknown status to PENDING", func() {
			status := mapCoinbaseStatus("UNKNOWN", "1.0", "0")
			Expect(status).To(Equal(models.ORDER_STATUS_PENDING))
		})
	})
})
