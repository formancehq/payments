package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Pool Deletion", func() {
	var (
		handlerFn http.HandlerFunc
		poolID    uuid.UUID
	)
	BeforeEach(func() {
		poolID = uuid.New()
	})

	Context("delete pool", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = poolsDelete(m)
		})

		It("should return a bad request error when poolID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "poolID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation delete err")
			m.EXPECT().PoolsDelete(gomock.Any(), gomock.Any()).Return(expectedErr)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "poolID", poolID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PoolsDelete(gomock.Any(), poolID).Return(nil)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "poolID", poolID.String()))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
