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

var _ = Describe("API v3 Conversions", func() {
	var (
		handlerFn    http.HandlerFunc
		conversionID models.ConversionID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		conversionID = models.ConversionID{Reference: "conv-ref", ConnectorID: connID}
	})

	Context("get conversions", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = conversionsGet(m)
		})

		It("should return an invalid ID error when conversion ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "conversionID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "conversionID", conversionID.String())
			m.EXPECT().ConversionsGet(gomock.Any(), conversionID).Return(
				&models.Conversion{}, fmt.Errorf("conversions get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "conversionID", conversionID.String())
			m.EXPECT().ConversionsGet(gomock.Any(), conversionID).Return(
				&models.Conversion{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
