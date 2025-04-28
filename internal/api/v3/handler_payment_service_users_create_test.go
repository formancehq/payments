package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payment Service Users Create", func() {
	var (
		handlerFn      http.HandlerFunc
		bankAccountIDs []string
	)
	BeforeEach(func() {
		bankAccountIDs = []string{uuid.New().String(), uuid.New().String()}
	})

	Context("create psu", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersCreate(m, validation.NewValidator())
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(psuReq PaymentServiceUsersCreateRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &psuReq))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("name missing", PaymentServiceUsersCreateRequest{}),
			Entry("name too long", PaymentServiceUsersCreateRequest{Name: generateTextString(1001)}),
			Entry("country invalid", PaymentServiceUsersCreateRequest{Name: "a", Address: &AddressRequest{Country: pointer.For("invalid")}}),
			Entry("phone number invalid", PaymentServiceUsersCreateRequest{Name: "a", ContactDetails: &ContactDetailsRequest{PhoneNumber: pointer.For("invalid")}}),
			Entry("email invalid", PaymentServiceUsersCreateRequest{Name: "a", ContactDetails: &ContactDetailsRequest{Email: pointer.For("invalid")}}),
			Entry("street number invalid", PaymentServiceUsersCreateRequest{Name: "a", Address: &AddressRequest{StreetNumber: pointer.For("invalid@")}}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("psu create err")
			m.EXPECT().PaymentServiceUsersCreate(gomock.Any(), gomock.Any()).Return(expectedErr)
			psuReq := PaymentServiceUsersCreateRequest{
				Name:           "reference",
				BankAccountIDs: bankAccountIDs,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &psuReq))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status created", func(ctx SpecContext) {
			m.EXPECT().PaymentServiceUsersCreate(gomock.Any(), gomock.Any()).Return(nil)
			psuReq := PaymentServiceUsersCreateRequest{
				Name:           "reference",
				BankAccountIDs: bankAccountIDs,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &psuReq))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})

		It("should return status created with optional fields", func(ctx SpecContext) {
			m.EXPECT().PaymentServiceUsersCreate(gomock.Any(), gomock.Any()).Return(nil)
			psuReq := PaymentServiceUsersCreateRequest{
				Name: "reference",
				ContactDetails: &ContactDetailsRequest{
					Email:       pointer.For("test@formance.com"),
					PhoneNumber: pointer.For("+3312131415"),
				},
				Address: &AddressRequest{
					StreetName:   pointer.For("test"),
					StreetNumber: pointer.For("1"),
					City:         pointer.For("test"),
					Region:       pointer.For("test"),
					PostalCode:   pointer.For("test"),
					Country:      pointer.For("FR"),
				},
				BankAccountIDs: bankAccountIDs,
				Metadata: map[string]string{
					"foo": "bar",
				},
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &psuReq))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})
	})
})
