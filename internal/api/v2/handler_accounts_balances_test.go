package v2

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Accounts Balances", func() {
	var (
		handlerFn http.HandlerFunc
		accID     models.AccountID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		accID = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
	})

	Context("list balances", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = accountsBalances(m)
		})

		It("should return a validation request error when account ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "accountID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "accountID", accID.String())
			m.EXPECT().BalancesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Balance]{}, fmt.Errorf("balances list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "accountID", accID.String())
			m.EXPECT().BalancesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Balance]{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
