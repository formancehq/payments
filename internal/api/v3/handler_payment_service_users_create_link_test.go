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

var _ = Describe("API v3 Payment Service Users Create Link", func() {
	var (
		handlerFn   http.HandlerFunc
		psuID       uuid.UUID
		connectorID models.ConnectorID
	)
	BeforeEach(func() {
		psuID = uuid.New()
		connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "test"}
	})

	Context("create link", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersCreateLink(m, validation.NewValidator())
		})

		It("should return an invalid ID error when psu ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodPost, "paymentServiceUserID", "invalidvalue", "connectorID", connectorID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an invalid ID error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodPost, "paymentServiceUserID", psuID.String(), "connectorID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an invalid ID error when idempotency key is invalid", func(ctx SpecContext) {
			req := prepareQueryRequestWithPath("/?Idempotency-Key=invalid", "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodPost, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		It("should return a bad request error when body is invalid JSON", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			req.Body = nil
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(linkReq PaymentServiceUserCreateLinkRequest) {
				req := prepareJSONRequest(http.MethodPost, &linkReq)
				req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
				handlerFn(w, req)
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("client redirect URL missing", PaymentServiceUserCreateLinkRequest{ApplicationName: "Test"}),
			Entry("client name missing", PaymentServiceUserCreateLinkRequest{ClientRedirectURL: "https://example.com/callback"}),
			Entry("client redirect URL invalid", PaymentServiceUserCreateLinkRequest{ApplicationName: "Test", ClientRedirectURL: "invalid-url"}),
			Entry("client redirect URL empty", PaymentServiceUserCreateLinkRequest{ApplicationName: "Test", ClientRedirectURL: ""}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserCreateLinkRequest{
				ApplicationName:   "Test",
				ClientRedirectURL: "https://example.com/callback",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			expectedErr := errors.New("create link error")
			m.EXPECT().PaymentServiceUsersCreateLink(gomock.Any(), "Test", psuID, connectorID, nil, gomock.Any()).Return(
				"", "", expectedErr,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return created status with link", func(ctx SpecContext) {
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserCreateLinkRequest{
				ApplicationName:   "Test",
				ClientRedirectURL: "https://example.com/callback",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			m.EXPECT().PaymentServiceUsersCreateLink(gomock.Any(), "Test", psuID, connectorID, nil, gomock.Any()).Return("test", "link", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusCreated, `{"attemptID":"test","link":"link"}`)
		})

		It("should return created status with link when idempotency key is provided", func(ctx SpecContext) {
			idempotencyKey := uuid.New()
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserCreateLinkRequest{
				ApplicationName:   "Test",
				ClientRedirectURL: "https://example.com/callback",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			req.URL.RawQuery = "Idempotency-Key=" + idempotencyKey.String()
			m.EXPECT().PaymentServiceUsersCreateLink(gomock.Any(), "Test", psuID, connectorID, &idempotencyKey, gomock.Any()).Return("test", "link", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusCreated, `{"attemptID":"test","link":"link"}`)
		})

		It("should handle empty idempotency key query parameter", func(ctx SpecContext) {
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserCreateLinkRequest{
				ApplicationName:   "Test",
				ClientRedirectURL: "https://example.com/callback",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			req.URL.RawQuery = "Idempotency-Key="
			m.EXPECT().PaymentServiceUsersCreateLink(gomock.Any(), "Test", psuID, connectorID, nil, gomock.Any()).Return("test", "link", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusCreated, `{"attemptID":"test","link":"link"}`)
		})

		It("should handle missing idempotency key query parameter", func(ctx SpecContext) {
			req := prepareJSONRequest(http.MethodPost, PaymentServiceUserCreateLinkRequest{
				ApplicationName:   "Test",
				ClientRedirectURL: "https://example.com/callback",
			})
			req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
			m.EXPECT().PaymentServiceUsersCreateLink(gomock.Any(), "Test", psuID, connectorID, nil, gomock.Any()).Return("test", "link", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusCreated, `{"attemptID":"test","link":"link"}`)
		})

		It("should handle various valid redirect URLs", func(ctx SpecContext) {
			validURLs := []string{
				"https://example.com/callback",
				"http://localhost:3000/redirect",
				"https://app.example.com/auth/callback?state=123",
				"https://test.example.com/oauth/callback#token=abc",
			}

			for _, url := range validURLs {
				w = httptest.NewRecorder()
				req := prepareJSONRequest(http.MethodPost, PaymentServiceUserCreateLinkRequest{
					ApplicationName:   "Test",
					ClientRedirectURL: url,
				})
				req = prepareQueryRequestWithBody(http.MethodPost, req.Body, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String())
				m.EXPECT().PaymentServiceUsersCreateLink(gomock.Any(), "Test", psuID, connectorID, nil, &url).Return("test", "link", nil)
				handlerFn(w, req)

				assertExpectedResponse(w.Result(), http.StatusCreated, `{"attemptID":"test","link":"link"}`)
			}
		})
	})
})
