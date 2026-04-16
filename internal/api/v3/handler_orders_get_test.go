package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Orders", func() {
	var (
		handlerFn http.HandlerFunc
		orderID   models.OrderID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		orderID = models.OrderID{Reference: "order-ref", ConnectorID: connID}
	})

	Context("get orders", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = ordersGet(m)
		})

		It("should return an invalid ID error when order ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "orderID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "orderID", orderID.String())
			m.EXPECT().OrdersGet(gomock.Any(), orderID).Return(
				&models.Order{}, fmt.Errorf("orders get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "orderID", orderID.String())
			m.EXPECT().OrdersGet(gomock.Any(), orderID).Return(
				&models.Order{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
