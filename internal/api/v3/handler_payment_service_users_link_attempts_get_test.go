package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payment Service Users Link Attempts Get", func() {
	var (
		handlerFn   http.HandlerFunc
		attemptID   uuid.UUID
		connectorID models.ConnectorID
	)
	BeforeEach(func() {
		attemptID = uuid.New()
		connectorID = models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "plaid",
		}
	})

	Context("get link attempt", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersLinkAttemptGet(m)
		})

		It("should return an invalid ID error when attempt ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet,
				"attemptID", "invalidvalue",
				"paymentServiceUserID", uuid.New().String(),
				"connectorID", connectorID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "attemptID", attemptID.String(), "paymentServiceUserID", uuid.New().String(), "connectorID", connectorID.String())
			expectedErr := errors.New("link attempt get error")
			m.EXPECT().PaymentServiceUsersLinkAttemptsGet(gomock.Any(), gomock.Any(), gomock.Any(), attemptID).Return(
				nil, expectedErr,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet,
				"attemptID", attemptID.String(),
				"paymentServiceUserID", uuid.New().String(),
				"connectorID", connectorID.String())
			attempt := &models.PSUOpenBankingConnectionAttempt{
				ID:     attemptID,
				PsuID:  uuid.New(),
				Status: models.PSUOpenBankingConnectionAttemptStatusPending,
			}
			m.EXPECT().PaymentServiceUsersLinkAttemptsGet(gomock.Any(), gomock.Any(), gomock.Any(), attemptID).Return(attempt, nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
