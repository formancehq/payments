package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Bank Accounts Create", func() {
	var (
		handlerFn     http.HandlerFunc
		accountNumber string
		iban          string
	)
	BeforeEach(func() {
		accountNumber = "1232434"
		iban = "DE89370400440532013000"
	})

	Context("create bank accounts", func() {
		var (
			w   *httptest.ResponseRecorder
			m   *backend.MockBackend
			bac bankAccountsCreateRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = bankAccountsCreate(m)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(bac bankAccountsCreateRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("account number missing", bankAccountsCreateRequest{}),
			Entry("iban missing", bankAccountsCreateRequest{AccountNumber: &accountNumber}),
			Entry("name missing", bankAccountsCreateRequest{AccountNumber: &accountNumber, IBAN: &iban}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("bank account create err")
			m.EXPECT().BankAccountsCreate(gomock.Any(), gomock.Any()).Return(expectedErr)
			bac = bankAccountsCreateRequest{
				Name:          "reference",
				IBAN:          &iban,
				AccountNumber: &accountNumber,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status created on success", func(ctx SpecContext) {
			m.EXPECT().BankAccountsCreate(gomock.Any(), gomock.Any()).Return(nil)
			bac = bankAccountsCreateRequest{
				Name:          "reference",
				IBAN:          &iban,
				AccountNumber: &accountNumber,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})
	})
})