package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v3 Payment Service Users Add Bank Account", func() {
	var (
		handlerFn     http.HandlerFunc
		bankAccountID uuid.UUID
		psuID         uuid.UUID
	)
	BeforeEach(func() {
		bankAccountID = uuid.New()
		psuID = uuid.New()
	})

	Context("PSU add bank account", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersAddBankAccount(m)
		})

		It("should return a bad request error when psu id is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a bad request error when bank account id is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", psuID.String(), "bankAccountID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("psu add account err")
			m.EXPECT().PaymentServiceUsersAddBankAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentServiceUserID", psuID.String(), "bankAccountID", bankAccountID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PaymentServiceUsersAddBankAccount(gomock.Any(), psuID, bankAccountID).Return(nil)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentServiceUserID", psuID.String(), "bankAccountID", bankAccountID.String()))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
