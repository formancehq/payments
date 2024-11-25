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

var _ = Describe("API v3 Accounts", func() {
	var (
		handlerFn http.HandlerFunc
		accID     models.AccountID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		accID = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
	})

	Context("get accounts", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = accountsGet(m)
		})

		It("should return an invalid ID error when account ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "accountID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "accountID", accID.String())
			m.EXPECT().AccountsGet(gomock.Any(), accID).Return(
				&models.Account{}, fmt.Errorf("accounts get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "accountID", accID.String())
			m.EXPECT().AccountsGet(gomock.Any(), accID).Return(
				&models.Account{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
