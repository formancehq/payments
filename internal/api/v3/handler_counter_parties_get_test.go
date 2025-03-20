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

var _ = Describe("API v3 Counter Parties", func() {
	var (
		handlerFn      http.HandlerFunc
		counterPartyID uuid.UUID
	)
	BeforeEach(func() {
		counterPartyID = uuid.New()
	})

	Context("get counter parties", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = counterPartiesGet(m)
		})

		It("should return an invalid ID error when counter party ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "counterPartyID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "counterPartyID", counterPartyID.String())
			m.EXPECT().CounterPartiesGet(gomock.Any(), counterPartyID).Return(
				&models.CounterParty{}, fmt.Errorf("counter parties get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "counterPartyID", counterPartyID.String())
			m.EXPECT().CounterPartiesGet(gomock.Any(), counterPartyID).Return(
				&models.CounterParty{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
