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

var _ = Describe("API v3 Get Pool", func() {
	var (
		handlerFn http.HandlerFunc
		poolID    uuid.UUID
	)
	BeforeEach(func() {
		poolID = uuid.New()
	})

	Context("get pools", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = poolsGet(m)
		})

		It("should return an invalid ID error when poolID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest("poolID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest("poolID", poolID.String())
			m.EXPECT().PoolsGet(gomock.Any(), poolID).Return(
				&models.Pool{}, fmt.Errorf("pool get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest("poolID", poolID.String())
			m.EXPECT().PoolsGet(gomock.Any(), poolID).Return(
				&models.Pool{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
