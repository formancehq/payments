package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Counter Parties Create", func() {
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
			cpr CounterPartiesCreateRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = counterPartiesCreate(m, validation.NewValidator())
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(bac CounterPartiesCreateRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &bac))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("name missing", CounterPartiesCreateRequest{}),
			Entry("account number & IBAN both missing", CounterPartiesCreateRequest{
				Name: "1",
				BankAccountInformation: &BankAccountInformationRequest{
					SwiftBicCode: pointer.For("test"),
				},
			}),
			Entry("name too long", CounterPartiesCreateRequest{Name: generateTextString(1001)}),
			Entry("country invalid", CounterPartiesCreateRequest{Name: "a", Address: &AddressRequest{Country: pointer.For("invalid")}}),
			Entry("iban invalid", CounterPartiesCreateRequest{
				Name: "1",
				BankAccountInformation: &BankAccountInformationRequest{
					IBAN:         pointer.For("FR12345678_%32"),
					SwiftBicCode: pointer.For("test"),
				},
			}),
			Entry("bic invalid", CounterPartiesCreateRequest{
				Name: "1",
				BankAccountInformation: &BankAccountInformationRequest{
					IBAN:         pointer.For("FR12345678_%32"),
					SwiftBicCode: pointer.For("aaaa$#@32"),
				},
			}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("counter party create err")
			m.EXPECT().CounterPartiesCreate(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			cpr = CounterPartiesCreateRequest{
				Name: "reference",
				BankAccountInformation: &BankAccountInformationRequest{
					IBAN: &iban,
				},
				ContactDetails: &ContactDetailsRequest{
					Email: pointer.For("test@formance.com"),
					Phone: pointer.For("0612345678"),
				},
				Address: &AddressRequest{
					StreetName:   pointer.For("street"),
					StreetNumber: pointer.For("1"),
					City:         pointer.For("city"),
					PostalCode:   pointer.For("1234"),
					Country:      pointer.For("FR"),
				},
				Metadata: map[string]string{
					"foo": "bar",
				},
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
			_ = accountNumber
		})

		It("should return status created with bank account information", func(ctx SpecContext) {
			m.EXPECT().CounterPartiesCreate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			cpr = CounterPartiesCreateRequest{
				Name: "reference",
				BankAccountInformation: &BankAccountInformationRequest{
					IBAN: &iban,
				},
				ContactDetails: &ContactDetailsRequest{
					Email: pointer.For("test@formance.com"),
					Phone: pointer.For("0612345678"),
				},
				Address: &AddressRequest{
					StreetName:   pointer.For("street"),
					StreetNumber: pointer.For("1"),
					City:         pointer.For("city"),
					PostalCode:   pointer.For("1234"),
					Country:      pointer.For("FR"),
				},
				Metadata: map[string]string{
					"foo": "bar",
				},
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})

		It("should return status created with bank account id instead of information", func(ctx SpecContext) {
			m.EXPECT().CounterPartiesCreate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			cpr = CounterPartiesCreateRequest{
				Name: "reference",
				BankAccountInformation: &BankAccountInformationRequest{
					BankAccountID: pointer.For(uuid.New().String()),
				},
				ContactDetails: &ContactDetailsRequest{
					Email: pointer.For("test@formance.com"),
					Phone: pointer.For("0612345678"),
				},
				Address: &AddressRequest{
					StreetName:   pointer.For("street"),
					StreetNumber: pointer.For("1"),
					City:         pointer.For("city"),
					PostalCode:   pointer.For("1234"),
					Country:      pointer.For("FR"),
				},
				Metadata: map[string]string{
					"foo": "bar",
				},
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})
	})
})
