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

var _ = Describe("API v3 Bank Accounts", func() {
	var (
		handlerFn http.HandlerFunc
		accID     uuid.UUID
	)
	BeforeEach(func() {
		accID = uuid.New()
	})

	Context("get bank accounts", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = bankAccountsGet(m)
		})

		It("should return an invalid ID error when account ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest("bankAccountID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest("bankAccountID", accID.String())
			m.EXPECT().BankAccountsGet(gomock.Any(), accID).Return(
				&models.BankAccount{}, fmt.Errorf("bank accounts get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest("bankAccountID", accID.String())
			m.EXPECT().BankAccountsGet(gomock.Any(), accID).Return(
				&models.BankAccount{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
