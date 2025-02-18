package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Pools Create", func() {
	var (
		handlerFn http.HandlerFunc
		accID     models.AccountID
		accID2    models.AccountID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		accID = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
		accID2 = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
	})

	Context("create pools", func() {
		var (
			w   *httptest.ResponseRecorder
			m   *backend.MockBackend
			cpr CreatePoolRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = poolsCreate(m, validation.NewValidator())
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(cpr CreatePoolRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("accountIDs missing", CreatePoolRequest{Name: "test"}),
			Entry("accountIDs invalid", CreatePoolRequest{Name: "test", AccountIDs: []string{"invalid"}}),
			Entry("name missing", CreatePoolRequest{AccountIDs: []string{accID.String()}}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment create err")
			m.EXPECT().PoolsCreate(gomock.Any(), gomock.Any()).Return(expectedErr)
			cpr = CreatePoolRequest{
				Name:       "name",
				AccountIDs: []string{accID.String()},
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status created on success", func(ctx SpecContext) {
			m.EXPECT().PoolsCreate(gomock.Any(), gomock.Any()).Return(nil)
			cpr = CreatePoolRequest{
				Name:       "name",
				AccountIDs: []string{accID.String(), accID2.String()},
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})
	})
})
