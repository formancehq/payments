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

var _ = Describe("API v3 Bank Accounts Update Metadata", func() {
	var (
		handlerFn     http.HandlerFunc
		bankAccountID uuid.UUID
	)
	BeforeEach(func() {
		bankAccountID = uuid.New()
	})

	Context("update bank account metadata", func() {
		var (
			w   *httptest.ResponseRecorder
			m   *backend.MockBackend
			bau BankAccountsUpdateMetadataRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = bankAccountsUpdateMetadata(m)
		})

		It("should return a bad request error when bank account is invalid", func(ctx SpecContext) {
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "bankAccountID", "invalid", &bau))

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		DescribeTable("validation errors",
			func(bau BankAccountsUpdateMetadataRequest) {
				handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "bankAccountID", bankAccountID.String(), &bau))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("metadata missing", BankAccountsUpdateMetadataRequest{}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("bank account update metadata err")
			m.EXPECT().BankAccountsUpdateMetadata(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			bau = BankAccountsUpdateMetadataRequest{
				Metadata: map[string]string{"meta": "data"},
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "bankAccountID", bankAccountID.String(), &bau))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			metadata := map[string]string{"meta": "data"}
			m.EXPECT().BankAccountsUpdateMetadata(gomock.Any(), gomock.Any(), metadata).Return(nil)
			bau = BankAccountsUpdateMetadataRequest{
				Metadata: metadata,
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "bankAccountID", bankAccountID.String(), &bau))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
