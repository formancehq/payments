package v2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Pools Balances At", func() {
	var (
		handlerFn http.HandlerFunc
		poolID    uuid.UUID
		now       time.Time
	)
	BeforeEach(func() {
		poolID = uuid.New()
		now = time.Now().UTC().Truncate(time.Second)
	})

	Context("pools balances at", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = poolsBalancesAt(m)
		})

		It("should return a validation request error when poolID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequestWithPath("/", "poolID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a validation request error when at param is missing", func(ctx SpecContext) {
			req := prepareQueryRequestWithPath("/", "poolID", poolID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			path := fmt.Sprintf("/?at=%s", now.Format(time.RFC3339))
			req := prepareQueryRequestWithPath(path, "poolID", poolID.String())
			m.EXPECT().PoolsBalancesAt(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				[]models.AggregatedBalance{},
				fmt.Errorf("balances list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a data object", func(ctx SpecContext) {
			path := fmt.Sprintf("/?at=%s", now.Format(time.RFC3339))
			req := prepareQueryRequestWithPath(path, "poolID", poolID.String())
			m.EXPECT().PoolsBalancesAt(gomock.Any(), poolID, now).Return(
				[]models.AggregatedBalance{},
				nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
