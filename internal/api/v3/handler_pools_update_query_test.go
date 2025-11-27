package v3

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

var _ = Describe("API v3 pools update query", func() {
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

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := prepareQueryRequestWithBody(http.MethodPatch, nil, "poolID", poolID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		It("should return a bad request error when poolID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequestWithBody(http.MethodPatch, nil, "poolID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("update error")
			m.EXPECT().PoolsUpdateQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			upr := PoolsUpdateQueryRequest{
				Query: map[string]any{
					"$match": map[string]any{
						"account_id": "123",
					},
				},
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "poolID", poolID.String(), &upr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PoolsUpdateQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			upr := PoolsUpdateQueryRequest{
				Query: map[string]any{
					"$match": map[string]any{
						"account_id": "123",
					},
				},
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "poolID", poolID.String(), &upr))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
