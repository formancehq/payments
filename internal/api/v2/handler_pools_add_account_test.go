package v2

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

var _ = Describe("API v2 pools add account", func() {
	var (
		handlerFn http.HandlerFunc
		accID     models.AccountID
		poolID    uuid.UUID
		paar      PoolsAddAccountRequest
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

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "poolID", poolID.String(), nil))

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
		})

		It("should return a bad request error when poolID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "poolID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("pool add account err")
			m.EXPECT().PoolsAddAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			paar = PoolsAddAccountRequest{
				AccountID: accID.String(),
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "poolID", poolID.String(), &paar))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PoolsAddAccount(gomock.Any(), poolID, accID).Return(nil)
			paar = PoolsAddAccountRequest{
				AccountID: accID.String(),
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "poolID", poolID.String(), &paar))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
