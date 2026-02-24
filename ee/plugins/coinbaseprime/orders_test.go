package coinbaseprime

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Plugin Orders", func() {
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
				"GBP":  2,
				"BTC":  8,
				"ETH":  18,
				"USDC": 6,
				"SOL":  9,
			},
			networkSymbols: map[string]string{},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next orders", func() {
		It("should return an error - ListOrders error", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ContainSubstring("test error")))
			Expect(resp).To(Equal(models.FetchNextOrdersResponse{}))
		})

		It("should fetch and convert a BUY order correctly", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:             "order-1",
							PortfolioID:    "portfolio-1",
							ProductID:      "BTC-USD",
							Side:           "BUY",
							Type:           "LIMIT",
							BaseQuantity:   "1.5",
							FilledQuantity: "0.5",
							Commission:     "10.50",
							LimitPrice:     "50000.00",
							Status:         "OPEN",
							TimeInForce:    "GTC",
							CreatedAt:      "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{
						NextCursor: "cursor-abc",
						HasNext:    true,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.HasMore).To(BeTrue())

			order := resp.Orders[0]
			Expect(order.Reference).To(Equal("order-1"))
			Expect(order.Direction).To(Equal(models.ORDER_DIRECTION_BUY))
			Expect(order.Type).To(Equal(models.ORDER_TYPE_LIMIT))
			Expect(order.SourceAsset).To(Equal("USD/2"))
			Expect(order.TargetAsset).To(Equal("BTC/8"))

			// BaseQuantityOrdered: 1.5 BTC = 150000000 (1.5 * 10^8)
			Expect(order.BaseQuantityOrdered).ToNot(BeNil())
			Expect(order.BaseQuantityOrdered.Cmp(big.NewInt(150000000))).To(Equal(0))

			// BaseQuantityFilled: 0.5 BTC = 50000000 (0.5 * 10^8)
			Expect(order.BaseQuantityFilled).ToNot(BeNil())
			Expect(order.BaseQuantityFilled.Cmp(big.NewInt(50000000))).To(Equal(0))

			// Fee: 10.50 USD = 1050 (10.50 * 10^2)
			Expect(order.Fee).ToNot(BeNil())
			Expect(order.Fee.Cmp(big.NewInt(1050))).To(Equal(0))

			// LimitPrice: 50000.00 USD = 5000000 (50000.00 * 10^2)
			Expect(order.LimitPrice).ToNot(BeNil())
			Expect(order.LimitPrice.Cmp(big.NewInt(5000000))).To(Equal(0))

			// Status: OPEN with filledQty > 0 && < baseQty → PARTIALLY_FILLED
			Expect(order.Status).To(Equal(models.ORDER_STATUS_PARTIALLY_FILLED))

			Expect(order.TimeInForce).To(Equal(models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED))

			// Verify pagination state
			var state ordersState
			Expect(resp.NewState).ToNot(BeNil())
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Cursor).To(Equal("cursor-abc"))
		})

		It("should map SELL direction correctly", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:             "order-sell",
							ProductID:      "BTC-USD",
							Side:           "SELL",
							Type:           "MARKET",
							BaseQuantity:   "2.0",
							FilledQuantity: "2.0",
							Status:         "FILLED",
							TimeInForce:    "GTC",
							CreatedAt:      "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders).To(HaveLen(1))

			order := resp.Orders[0]
			Expect(order.Direction).To(Equal(models.ORDER_DIRECTION_SELL))
			// SELL BTC-USD: source=BTC (what you spend), target=USD (what you receive)
			Expect(order.SourceAsset).To(Equal("BTC/8"))
			Expect(order.TargetAsset).To(Equal("USD/2"))
		})

		It("should map FILLED status", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:             "order-filled",
							ProductID:      "BTC-USD",
							Side:           "BUY",
							Type:           "LIMIT",
							BaseQuantity:   "1.0",
							FilledQuantity: "1.0",
							Status:         "FILLED",
							TimeInForce:    "GTC",
							CreatedAt:      "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_FILLED))
		})

		It("should map CANCELLED status", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-cancelled",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "LIMIT",
							BaseQuantity: "1.0",
							Status:       "CANCELLED",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_CANCELLED))
		})

		It("should map EXPIRED status", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-expired",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "LIMIT",
							BaseQuantity: "1.0",
							Status:       "EXPIRED",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_EXPIRED))
		})

		It("should map FAILED status", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-failed",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "MARKET",
							BaseQuantity: "1.0",
							Status:       "FAILED",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_FAILED))
		})

		It("should map PENDING status", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-pending",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "LIMIT",
							BaseQuantity: "1.0",
							Status:       "PENDING",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_PENDING))
		})

		It("should detect PARTIALLY_FILLED from OPEN", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:             "order-partial",
							ProductID:      "BTC-USD",
							Side:           "BUY",
							Type:           "LIMIT",
							BaseQuantity:   "10.0",
							FilledQuantity: "3.0",
							Status:         "OPEN",
							TimeInForce:    "GTC",
							CreatedAt:      "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].Status).To(Equal(models.ORDER_STATUS_PARTIALLY_FILLED))
		})

		It("should map MARKET type", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-market",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "MARKET",
							BaseQuantity: "1.0",
							Status:       "FILLED",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].Type).To(Equal(models.ORDER_TYPE_MARKET))
		})

		It("should map STOP_LIMIT type", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-stop-limit",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "STOP_LIMIT",
							BaseQuantity: "1.0",
							Status:       "PENDING",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].Type).To(Equal(models.ORDER_TYPE_STOP_LIMIT))
		})

		It("should map IOC time in force", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-ioc",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "LIMIT",
							BaseQuantity: "1.0",
							Status:       "FILLED",
							TimeInForce:  "IOC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].TimeInForce).To(Equal(models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL))
		})

		It("should map FOK time in force", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-fok",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "LIMIT",
							BaseQuantity: "1.0",
							Status:       "FILLED",
							TimeInForce:  "FOK",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders[0].TimeInForce).To(Equal(models.TIME_IN_FORCE_FILL_OR_KILL))
		})

		It("should fail on invalid product ID", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-bad-product",
							ProductID:    "INVALID",
							Side:         "BUY",
							Type:         "MARKET",
							BaseQuantity: "1.0",
							Status:       "FILLED",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ContainSubstring("invalid product ID")))
			Expect(resp).To(Equal(models.FetchNextOrdersResponse{}))
		})

		It("should fail on unsupported base asset", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-bad-base",
							ProductID:    "UNKNOWN-USD",
							Side:         "BUY",
							Type:         "MARKET",
							BaseQuantity: "1.0",
							Status:       "FILLED",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ContainSubstring("unsupported base asset")))
			Expect(resp).To(Equal(models.FetchNextOrdersResponse{}))
		})

		It("should fail on unsupported quote asset", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-bad-quote",
							ProductID:    "BTC-UNKNOWN",
							Side:         "BUY",
							Type:         "MARKET",
							BaseQuantity: "1.0",
							Status:       "FILLED",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ContainSubstring("unsupported quote asset")))
			Expect(resp).To(Equal(models.FetchNextOrdersResponse{}))
		})

		It("should fail on invalid CreatedAt", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-bad-date",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "MARKET",
							BaseQuantity: "1.0",
							Status:       "FILLED",
							TimeInForce:  "GTC",
							CreatedAt:    "invalid",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ContainSubstring("failed to parse order createdAt")))
			Expect(resp).To(Equal(models.FetchNextOrdersResponse{}))
		})

		It("should use cursor from state", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{"cursor":"existing-cursor"}`),
				PageSize: 25,
			}

			m.EXPECT().ListOrders(gomock.Any(), "existing-cursor", 25).Return(
				&client.OrdersResponse{
					Orders:     []client.Order{},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should handle empty commission", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:           "order-no-fee",
							ProductID:    "BTC-USD",
							Side:         "BUY",
							Type:         "MARKET",
							BaseQuantity: "1.0",
							Commission:   "",
							Status:       "FILLED",
							TimeInForce:  "GTC",
							CreatedAt:    "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders).To(HaveLen(1))
			Expect(resp.Orders[0].Fee).To(BeNil())
		})

		It("should handle empty quantities", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().ListOrders(gomock.Any(), "", 10).Return(
				&client.OrdersResponse{
					Orders: []client.Order{
						{
							ID:             "order-empty-qty",
							ProductID:      "BTC-USD",
							Side:           "BUY",
							Type:           "MARKET",
							BaseQuantity:   "",
							FilledQuantity: "",
							Status:         "PENDING",
							TimeInForce:    "GTC",
							CreatedAt:      "2024-01-15T10:30:00Z",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Orders).To(HaveLen(1))

			order := resp.Orders[0]
			Expect(order.BaseQuantityOrdered).ToNot(BeNil())
			Expect(order.BaseQuantityOrdered.Cmp(big.NewInt(0))).To(Equal(0))
			Expect(order.BaseQuantityFilled).ToNot(BeNil())
			Expect(order.BaseQuantityFilled.Cmp(big.NewInt(0))).To(Equal(0))
		})
	})
})
