package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v3 pools add account", func() {
	var (
		handlerFn http.HandlerFunc
		accID     models.AccountID
		poolID    uuid.UUID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		accID = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
		poolID = uuid.New()
	})

	Context("pool add account", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = poolsAddAccount(m)
		})

		It("should return a bad request error when poolID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "poolID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a bad request error when accountID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "poolID", poolID.String(), "accountID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("pool add account err")
			m.EXPECT().PoolsAddAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "poolID", poolID.String(), "accountID", accID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PoolsAddAccount(gomock.Any(), poolID, accID).Return(nil)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "poolID", poolID.String(), "accountID", accID.String()))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
