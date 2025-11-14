package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
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
			bac BankAccountsCreateRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = bankAccountsCreate(m, validation.NewValidator())
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(bac BankAccountsCreateRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("account number & IBAN both missing", BankAccountsCreateRequest{Name: "1"}),
			Entry("name missing", BankAccountsCreateRequest{AccountNumber: &accountNumber, IBAN: &iban}),
			Entry("name too long", BankAccountsCreateRequest{Name: generateTextString(1001), IBAN: &iban}),
			Entry("country invalid", BankAccountsCreateRequest{Name: "a", IBAN: &iban, Country: pointer.For("invalid")}),
			Entry("iban invalid", BankAccountsCreateRequest{Name: "a", IBAN: pointer.For("FR12345678_%32"), Country: pointer.For("DE")}),
			Entry("bic invalid", BankAccountsCreateRequest{Name: "a", IBAN: &iban, SwiftBicCode: pointer.For("aaaa$#@32"), Country: pointer.For("DE")}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("bank account create err")
			m.EXPECT().BankAccountsCreate(gomock.Any(), gomock.Any()).Return(expectedErr)
			bac = BankAccountsCreateRequest{
				Name:          "reference",
				IBAN:          &iban,
				AccountNumber: &accountNumber,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status created with IBAN only", func(ctx SpecContext) {
			m.EXPECT().BankAccountsCreate(gomock.Any(), gomock.Any()).Return(nil)
			bac = BankAccountsCreateRequest{
				Name: "reference",
				IBAN: &iban,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})

		It("should return status created with account number only", func(ctx SpecContext) {
			m.EXPECT().BankAccountsCreate(gomock.Any(), gomock.Any()).Return(nil)
			bac = BankAccountsCreateRequest{
				Name:          "reference",
				AccountNumber: &accountNumber,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})

		It("should return status created with optional fields", func(ctx SpecContext) {
			m.EXPECT().BankAccountsCreate(gomock.Any(), gomock.Any()).Return(nil)
			bac = BankAccountsCreateRequest{
				Name:          "some name",
				AccountNumber: &accountNumber,
				Country:       pointer.For("FR"),
				SwiftBicCode:  pointer.For("HBUKGB4B"),
				Metadata:      map[string]string{"greeting": "hi"},
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})
	})
})
