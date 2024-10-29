package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Accounts Create", func() {
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
			cra createAccountRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = accountsCreate(m)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(cra createAccountRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &cra))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", createAccountRequest{}),
			Entry("connectorID missing", createAccountRequest{Reference: "reference"}),
			Entry("createdAt missing", createAccountRequest{Reference: "reference", ConnectorID: "id"}),
			Entry("accountName missing", createAccountRequest{Reference: "reference", ConnectorID: "id", CreatedAt: time.Now()}),
			Entry("type missing", createAccountRequest{
				Reference: "reference", ConnectorID: "id", CreatedAt: time.Now(), AccountName: "accountName",
			}),
			Entry("connectorID invalid", createAccountRequest{
				Reference: "reference", ConnectorID: "id", CreatedAt: time.Now(), AccountName: "accountName", Type: "type",
			}),
			Entry("type invalid", createAccountRequest{
				Reference: "reference", ConnectorID: connID.String(), CreatedAt: time.Now(), AccountName: "accountName", Type: "type",
			}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("account create err")
			m.EXPECT().AccountsCreate(gomock.Any(), gomock.Any()).Return(expectedErr)
			cra = createAccountRequest{
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
			cra = createAccountRequest{
				Reference:   "reference",
				ConnectorID: connID.String(),
				CreatedAt:   time.Now(),
				AccountName: "accountName",
				Type:        string(models.ACCOUNT_TYPE_EXTERNAL),
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cra))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})
	})
})
