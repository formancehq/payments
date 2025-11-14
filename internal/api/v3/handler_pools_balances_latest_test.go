package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v3 Pools Balances Latest", func() {
	var (
		handlerFn http.HandlerFunc
		poolID    uuid.UUID
	)
	BeforeEach(func() {
		poolID = uuid.New()
	})

	Context("pools balances latest", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = poolsBalancesLatest(m)
		})

		It("should return a validation request error when poolID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequestWithPath("/", "poolID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequestWithPath("/", "poolID", poolID.String())
			m.EXPECT().PoolsBalances(gomock.Any(), gomock.Any()).Return(
				[]models.AggregatedBalance{},
				fmt.Errorf("balances list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a data object", func(ctx SpecContext) {
			req := prepareQueryRequestWithPath("/", "poolID", poolID.String())
			m.EXPECT().PoolsBalances(gomock.Any(), poolID).Return(
				[]models.AggregatedBalance{},
				nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
