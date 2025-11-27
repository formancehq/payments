package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payment Service Users Link Attempts List", func() {
	var (
		handlerFn   http.HandlerFunc
		psuID       uuid.UUID
		connectorID models.ConnectorID
	)
	BeforeEach(func() {
		psuID = uuid.New()
		connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "test"}
	})

	Context("list link attempts", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersLinkAttemptList(m)
		})

		It("should return an invalid ID error when psu ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", "invalidvalue", "connectorID", connectorID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an invalid ID error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", psuID.String(), "connectorID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a validation error when pagination is invalid", func(ctx SpecContext) {
			req := prepareQueryRequestWithPath("/?pageSize=invalid", "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			expectedErr := errors.New("link attempts list error")
			m.EXPECT().PaymentServiceUsersLinkAttemptsList(gomock.Any(), psuID, connectorID, gomock.Any()).Return(
				&bunpaginate.Cursor[models.OpenBankingConnectionAttempt]{}, expectedErr,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			cursor := &bunpaginate.Cursor[models.OpenBankingConnectionAttempt]{
				Data: []models.OpenBankingConnectionAttempt{
					{
						ID:     uuid.New(),
						PsuID:  psuID,
						Status: models.OpenBankingConnectionAttemptStatusPending,
					},
				},
			}
			m.EXPECT().PaymentServiceUsersLinkAttemptsList(gomock.Any(), psuID, connectorID, gomock.Any()).Return(cursor, nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
