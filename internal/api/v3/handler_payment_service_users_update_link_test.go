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

var _ = Describe("API v3 Payment Service Users Update Link", func() {
	var (
		handlerFn    http.HandlerFunc
		psuID        uuid.UUID
		connectorID  models.ConnectorID
		connectionID string
	)
	BeforeEach(func() {
		psuID = uuid.New()
		connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "test"}
		connectionID = "test-connection-id"
	})

	Context("update link", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersUpdateLink(m, validation.NewValidator())
		})

		It("should return an invalid ID error when psu ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodPost, "paymentServiceUserID", "invalidvalue", "connectorID", connectorID.String(), "connectionID", connectionID)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an invalid ID error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodPost, "paymentServiceUserID", psuID.String(), "connectorID", "invalidvalue", "connectionID", connectionID)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an invalid ID error when idempotency key is invalid", func(ctx SpecContext) {
			req := prepareQueryRequestWithPath("/?Idempotency-Key=invalid", "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodPost, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		It("should return a validation error when client redirect URL is missing", func(ctx SpecContext) {
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserUpdateLinkRequest{})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
		})

		It("should return a validation error when client redirect URL is invalid", func(ctx SpecContext) {
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserUpdateLinkRequest{
				ClientRedirectURL: "invalid-url",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserUpdateLinkRequest{
				ClientRedirectURL: "https://example.com/callback",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			expectedErr := errors.New("update link error")
			m.EXPECT().PaymentServiceUsersUpdateLink(gomock.Any(), psuID, connectorID, connectionID, gomock.Any(), gomock.Any()).Return(
				"", "", expectedErr,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return accepted status with task ID", func(ctx SpecContext) {
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserUpdateLinkRequest{
				ClientRedirectURL: "https://example.com/callback",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			m.EXPECT().PaymentServiceUsersUpdateLink(gomock.Any(), psuID, connectorID, connectionID, gomock.Any(), gomock.Any()).Return("test", "link", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusAccepted, `{"attemptID":"test","link":"link"}`)
		})

		It("should return accepted status with task ID when idempotency key is provided", func(ctx SpecContext) {
			idempotencyKey := uuid.New()
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserUpdateLinkRequest{
				ClientRedirectURL: "https://example.com/callback",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			req.URL.RawQuery = "Idempotency-Key=" + idempotencyKey.String()
			m.EXPECT().PaymentServiceUsersUpdateLink(gomock.Any(), psuID, connectorID, connectionID, &idempotencyKey, gomock.Any()).Return("test", "link", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusAccepted, `{"attemptID":"test","link":"link"}`)
		})
	})
})
