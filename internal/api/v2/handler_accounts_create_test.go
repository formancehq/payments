package v2

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Accounts Create", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("create accounts", func() {
		var (
			w   *httptest.ResponseRecorder
			m   *backend.MockBackend
			cra CreateAccountRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = accountsCreate(m, validation.NewValidator())
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(cra CreateAccountRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &cra))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", CreateAccountRequest{}),
			Entry("connectorID missing", CreateAccountRequest{Reference: "reference"}),
			Entry("createdAt missing", CreateAccountRequest{Reference: "reference", ConnectorID: "id"}),
			Entry("accountName missing", CreateAccountRequest{Reference: "reference", ConnectorID: "id", CreatedAt: time.Now()}),
			Entry("type missing", CreateAccountRequest{
				Reference: "reference", ConnectorID: "id", CreatedAt: time.Now(), AccountName: "accountName",
			}),
			Entry("connectorID invalid", CreateAccountRequest{
				Reference: "reference", ConnectorID: "id", CreatedAt: time.Now(), AccountName: "accountName", Type: "type",
			}),
			Entry("type invalid", CreateAccountRequest{
				Reference: "reference", ConnectorID: connID.String(), CreatedAt: time.Now(), AccountName: "accountName", Type: "type",
			}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("account create err")
			m.EXPECT().AccountsCreate(gomock.Any(), gomock.Any()).Return(expectedErr)
			cra = CreateAccountRequest{
				Reference:   "reference",
				ConnectorID: connID.String(),
				CreatedAt:   time.Now(),
				AccountName: "accountName",
				Type:        string(models.ACCOUNT_TYPE_EXTERNAL),
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cra))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status created on success", func(ctx SpecContext) {
			m.EXPECT().AccountsCreate(gomock.Any(), gomock.Any()).Return(nil)
			cra = CreateAccountRequest{
				Reference:   "reference",
				ConnectorID: connID.String(),
				CreatedAt:   time.Now(),
				AccountName: "accountName",
				Type:        string(models.ACCOUNT_TYPE_EXTERNAL),
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cra))
			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
