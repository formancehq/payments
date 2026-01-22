//go:build it

package test_suite

import (
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// CreateOrderRequest matches the API request structure
type CreateOrderRequest struct {
	Reference           string            `json:"reference"`
	ConnectorID         string            `json:"connectorID"`
	Direction           string            `json:"direction"`
	SourceAsset         string            `json:"sourceAsset"`
	TargetAsset         string            `json:"targetAsset"`
	Type                string            `json:"type"`
	BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
	LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
	StopPrice           *big.Int          `json:"stopPrice,omitempty"`
	TimeInForce         string            `json:"timeInForce,omitempty"`
	ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	SkipValidation      bool              `json:"skipValidation,omitempty"`
	SendToExchange      *bool             `json:"sendToExchange,omitempty"`
	WaitResult          bool              `json:"waitResult,omitempty"`
}

// CreateOrderResponse matches the API response structure
type CreateOrderResponse struct {
	Order struct {
		ID                  string            `json:"id"`
		Reference           string            `json:"reference"`
		ConnectorID         string            `json:"connectorID"`
		Direction           string            `json:"direction"`
		SourceAsset         string            `json:"sourceAsset"`
		TargetAsset         string            `json:"targetAsset"`
		Type                string            `json:"type"`
		Status              string            `json:"status"`
		BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
		LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
		StopPrice           *big.Int          `json:"stopPrice,omitempty"`
		TimeInForce         string            `json:"timeInForce"`
		ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
		Metadata            map[string]string `json:"metadata,omitempty"`
		CreatedAt           time.Time         `json:"createdAt"`
		UpdatedAt           time.Time         `json:"updatedAt"`
	} `json:"order"`
	Warnings []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"warnings,omitempty"`
	TaskID *string `json:"taskID,omitempty"`
}

// OrderResponse matches the get order API response structure
type OrderResponse struct {
	Data struct {
		ID                  string            `json:"id"`
		Reference           string            `json:"reference"`
		ConnectorID         string            `json:"connectorID"`
		Direction           string            `json:"direction"`
		SourceAsset         string            `json:"sourceAsset"`
		TargetAsset         string            `json:"targetAsset"`
		Type                string            `json:"type"`
		Status              string            `json:"status"`
		BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
		LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
		TimeInForce         string            `json:"timeInForce"`
		Metadata            map[string]string `json:"metadata,omitempty"`
		CreatedAt           time.Time         `json:"createdAt"`
		UpdatedAt           time.Time         `json:"updatedAt"`
	} `json:"data"`
}

// OrdersListResponse matches the list orders API response structure
type OrdersListResponse struct {
	Cursor struct {
		PageSize int `json:"pageSize"`
		HasMore  bool `json:"hasMore"`
		Data     []struct {
			ID                  string            `json:"id"`
			Reference           string            `json:"reference"`
			ConnectorID         string            `json:"connectorID"`
			Direction           string            `json:"direction"`
			SourceAsset         string            `json:"sourceAsset"`
			TargetAsset         string            `json:"targetAsset"`
			Type                string            `json:"type"`
			Status              string            `json:"status"`
			BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
			BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
			LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
			TimeInForce         string            `json:"timeInForce"`
			Metadata            map[string]string `json:"metadata,omitempty"`
			CreatedAt           time.Time         `json:"createdAt"`
			UpdatedAt           time.Time         `json:"updatedAt"`
		} `json:"data"`
	} `json:"cursor"`
}

var _ = Context("Payments API Orders", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		app *deferred.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                      stack,
			PostgresConfiguration:      db.GetValue().ConnectionOptions(),
			NatsURL:                    natsServer.GetValue().ClientURL(),
			TemporalNamespace:          temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:            temporalServer.GetValue().Address(),
			Output:                     GinkgoWriter,
			SkipOutboxScheduleCreation: true,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("creating and managing orders", func() {
		var (
			connectorID string
			err         error
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should create a LIMIT BUY order successfully", func() {
			sendToExchange := false
			createRequest := CreateOrderRequest{
				Reference:           "test-order-" + uuid.New().String()[:8],
				ConnectorID:         connectorID,
				Direction:           "BUY",
				SourceAsset:         "USD/2",
				TargetAsset:         "BTC/8",
				Type:                "LIMIT",
				BaseQuantityOrdered: big.NewInt(100000000),
				LimitPrice:          big.NewInt(5000000000000),
				TimeInForce:         "GTC",
				Metadata:            map[string]string{"test": "value"},
				SkipValidation:      true,
				SendToExchange:      &sendToExchange,
			}

			var createResponse CreateOrderResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/orders", createRequest, &createResponse)
			Expect(err).To(BeNil())
			Expect(createResponse.Order.Reference).To(Equal(createRequest.Reference))
			Expect(createResponse.Order.Status).To(Equal("PENDING"))
			Expect(createResponse.Order.Direction).To(Equal("BUY"))
			Expect(createResponse.Order.Type).To(Equal("LIMIT"))
			Expect(createResponse.Order.TimeInForce).To(Equal("GOOD_UNTIL_CANCELLED"))
		})

		It("should create a MARKET SELL order successfully", func() {
			sendToExchange := false
			createRequest := CreateOrderRequest{
				Reference:           "test-market-order-" + uuid.New().String()[:8],
				ConnectorID:         connectorID,
				Direction:           "SELL",
				SourceAsset:         "BTC/8",
				TargetAsset:         "USD/2",
				Type:                "MARKET",
				BaseQuantityOrdered: big.NewInt(50000000),
				TimeInForce:         "IOC",
				SkipValidation:      true,
				SendToExchange:      &sendToExchange,
			}

			var createResponse CreateOrderResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/orders", createRequest, &createResponse)
			Expect(err).To(BeNil())
			Expect(createResponse.Order.Reference).To(Equal(createRequest.Reference))
			Expect(createResponse.Order.Status).To(Equal("PENDING"))
			Expect(createResponse.Order.Direction).To(Equal("SELL"))
			Expect(createResponse.Order.Type).To(Equal("MARKET"))
			Expect(createResponse.Order.TimeInForce).To(Equal("IMMEDIATE_OR_CANCEL"))
		})

		It("should get an order by ID", func() {
			// First create an order
			sendToExchange := false
			createRequest := CreateOrderRequest{
				Reference:           "test-get-order-" + uuid.New().String()[:8],
				ConnectorID:         connectorID,
				Direction:           "BUY",
				SourceAsset:         "USD/2",
				TargetAsset:         "ETH/8",
				Type:                "LIMIT",
				BaseQuantityOrdered: big.NewInt(200000000),
				LimitPrice:          big.NewInt(300000000000),
				TimeInForce:         "GTC",
				SkipValidation:      true,
				SendToExchange:      &sendToExchange,
			}

			var createResponse CreateOrderResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/orders", createRequest, &createResponse)
			Expect(err).To(BeNil())

			// Now get the order
			orderID := createResponse.Order.ID
			var getResponse OrderResponse
			err = app.GetValue().Client().Get(ctx, "/v3/orders/"+orderID, &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.ID).To(Equal(orderID))
			Expect(getResponse.Data.Reference).To(Equal(createRequest.Reference))
			Expect(getResponse.Data.ConnectorID).To(Equal(connectorID))
		})

		It("should list orders with pagination", func() {
			// Create multiple orders
			sendToExchange := false
			for i := 0; i < 3; i++ {
				createRequest := CreateOrderRequest{
					Reference:           "test-list-order-" + uuid.New().String()[:8],
					ConnectorID:         connectorID,
					Direction:           "BUY",
					SourceAsset:         "USD/2",
					TargetAsset:         "BTC/8",
					Type:                "LIMIT",
					BaseQuantityOrdered: big.NewInt(int64(100000000 * (i + 1))),
					LimitPrice:          big.NewInt(5000000000000),
					TimeInForce:         "GTC",
					SkipValidation:      true,
					SendToExchange:      &sendToExchange,
				}

				var createResponse CreateOrderResponse
				err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/orders", createRequest, &createResponse)
				Expect(err).To(BeNil())
			}

			// List orders
			var listResponse OrdersListResponse
			err := app.GetValue().Client().Get(ctx, "/v3/orders", &listResponse)
			Expect(err).To(BeNil())
			Expect(len(listResponse.Cursor.Data)).To(BeNumerically(">=", 3))
		})

		It("should reject LIMIT order without limitPrice", func() {
			sendToExchange := false
			createRequest := CreateOrderRequest{
				Reference:           "test-invalid-order-" + uuid.New().String()[:8],
				ConnectorID:         connectorID,
				Direction:           "BUY",
				SourceAsset:         "USD/2",
				TargetAsset:         "BTC/8",
				Type:                "LIMIT",
				BaseQuantityOrdered: big.NewInt(100000000),
				// Missing LimitPrice for LIMIT order
				TimeInForce:    "GTC",
				SkipValidation: true,
				SendToExchange: &sendToExchange,
			}

			var createResponse CreateOrderResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/orders", createRequest, &createResponse)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("limitPrice is required for LIMIT orders"))
		})

		It("should reject GTD order without expiresAt", func() {
			sendToExchange := false
			createRequest := CreateOrderRequest{
				Reference:           "test-gtd-order-" + uuid.New().String()[:8],
				ConnectorID:         connectorID,
				Direction:           "BUY",
				SourceAsset:         "USD/2",
				TargetAsset:         "BTC/8",
				Type:                "LIMIT",
				BaseQuantityOrdered: big.NewInt(100000000),
				LimitPrice:          big.NewInt(5000000000000),
				TimeInForce:         "GTD",
				// Missing ExpiresAt for GTD order
				SkipValidation: true,
				SendToExchange: &sendToExchange,
			}

			var createResponse CreateOrderResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/orders", createRequest, &createResponse)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("expiresAt is required for GTD"))
		})

		It("should create GTD order with expiresAt", func() {
			sendToExchange := false
			expiresAt := time.Now().Add(24 * time.Hour)
			createRequest := CreateOrderRequest{
				Reference:           "test-gtd-valid-order-" + uuid.New().String()[:8],
				ConnectorID:         connectorID,
				Direction:           "BUY",
				SourceAsset:         "USD/2",
				TargetAsset:         "BTC/8",
				Type:                "LIMIT",
				BaseQuantityOrdered: big.NewInt(100000000),
				LimitPrice:          big.NewInt(5000000000000),
				TimeInForce:         "GTD",
				ExpiresAt:           &expiresAt,
				SkipValidation:      true,
				SendToExchange:      &sendToExchange,
			}

			var createResponse CreateOrderResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/orders", createRequest, &createResponse)
			Expect(err).To(BeNil())
			Expect(createResponse.Order.TimeInForce).To(Equal("GOOD_UNTIL_DATE_TIME"))
			Expect(createResponse.Order.ExpiresAt).NotTo(BeNil())
		})
	})
})
