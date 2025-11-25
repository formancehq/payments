package v2

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 pools update query", func() {
	var (
		handlerFn http.HandlerFunc
		poolID    uuid.UUID
	)
	BeforeEach(func() {
		poolID = uuid.New()
	})

	Context("pools update query", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = poolsUpdateQuery(m, validation.NewValidator())
		})

		It("should return a bad request error when poolID is invalid", func(ctx SpecContext) {
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "poolID", "invalid", &map[string]any{}))
			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("pools update query err")
			m.EXPECT().PoolsUpdateQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "poolID", poolID.String(), &map[string]any{}))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PoolsUpdateQuery(gomock.Any(), poolID, gomock.Any()).Return(nil)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "poolID", poolID.String(), &map[string]any{}))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
