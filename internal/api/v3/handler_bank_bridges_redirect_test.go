package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Bank Bridges Redirect", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
	)

	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("bank bridges redirect", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)

		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = bankBridgesRedirect(m)
		})

		It("should return a bad request error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a bad request error when body cannot be read", func(ctx SpecContext) {
			// Create a request with a body that cannot be read
			req := prepareQueryRequest(http.MethodPost, "connectorID", connID.String())
			req.Body = &errorReader{}

			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		It("should handle empty body successfully", func(ctx SpecContext) {
			expectedRedirectURL := "https://example.com/redirect"
			m.EXPECT().PaymentServiceUsersCompleteLinkFlow(
				gomock.Any(),
				connID,
				models.HTTPCallInformation{
					QueryValues: url.Values{},
					Headers:     http.Header{},
					Body:        []byte{},
				},
			).Return(expectedRedirectURL, nil)

			req := prepareQueryRequest(http.MethodPost, "connectorID", connID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusTemporaryRedirect, "")
			Expect(w.Header().Get("Location")).To(Equal(expectedRedirectURL))
		})

		It("should handle request with query parameters and headers", func(ctx SpecContext) {
			expectedRedirectURL := "https://example.com/redirect"
			queryParams := url.Values{
				"state": []string{"test-state"},
				"code":  []string{"test-code"},
			}
			headers := http.Header{
				"Authorization": []string{"Bearer token"},
				"Content-Type":  []string{"application/json"},
			}
			body := []byte(`{"test": "data"}`)

			m.EXPECT().PaymentServiceUsersCompleteLinkFlow(
				gomock.Any(),
				connID,
				models.HTTPCallInformation{
					QueryValues: queryParams,
					Headers:     headers,
					Body:        body,
				},
			).Return(expectedRedirectURL, nil)

			req := prepareQueryRequestWithBody(http.MethodPost, strings.NewReader(`{"test": "data"}`), "connectorID", connID.String())
			req.URL.RawQuery = "state=test-state&code=test-code"
			req.Header = headers

			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusTemporaryRedirect, "")
			Expect(w.Header().Get("Location")).To(Equal(expectedRedirectURL))
		})

		It("should return no content when noRedirect query parameter is true", func(ctx SpecContext) {
			expectedRedirectURL := "https://example.com/redirect"
			queryParams := url.Values{
				models.NoRedirectQueryParamID: []string{"true"},
			}

			m.EXPECT().PaymentServiceUsersCompleteLinkFlow(
				gomock.Any(),
				connID,
				models.HTTPCallInformation{
					QueryValues: queryParams,
					Headers:     http.Header{},
					Body:        []byte{},
				},
			).Return(expectedRedirectURL, nil)

			req := prepareQueryRequest(http.MethodPost, "connectorID", connID.String())
			req.URL.RawQuery = "noRedirect=true"
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})

		It("should redirect when noRedirect query parameter is false", func(ctx SpecContext) {
			expectedRedirectURL := "https://example.com/redirect"
			queryParams := url.Values{
				models.NoRedirectQueryParamID: []string{"false"},
			}

			m.EXPECT().PaymentServiceUsersCompleteLinkFlow(
				gomock.Any(),
				connID,
				models.HTTPCallInformation{
					QueryValues: queryParams,
					Headers:     http.Header{},
					Body:        []byte{},
				},
			).Return(expectedRedirectURL, nil)

			req := prepareQueryRequest(http.MethodPost, "connectorID", connID.String())
			req.URL.RawQuery = "noRedirect=false"
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusTemporaryRedirect, "")
			Expect(w.Header().Get("Location")).To(Equal(expectedRedirectURL))
		})

		It("should redirect when noRedirect query parameter is not present", func(ctx SpecContext) {
			expectedRedirectURL := "https://example.com/redirect"

			m.EXPECT().PaymentServiceUsersCompleteLinkFlow(
				gomock.Any(),
				connID,
				models.HTTPCallInformation{
					QueryValues: url.Values{},
					Headers:     http.Header{},
					Body:        []byte{},
				},
			).Return(expectedRedirectURL, nil)

			req := prepareQueryRequest(http.MethodPost, "connectorID", connID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusTemporaryRedirect, "")
			Expect(w.Header().Get("Location")).To(Equal(expectedRedirectURL))
		})

		It("should handle backend error and return internal server error", func(ctx SpecContext) {
			expectedErr := errors.New("backend error")
			m.EXPECT().PaymentServiceUsersCompleteLinkFlow(
				gomock.Any(),
				connID,
				models.HTTPCallInformation{
					QueryValues: url.Values{},
					Headers:     http.Header{},
					Body:        []byte{},
				},
			).Return("", expectedErr)

			req := prepareQueryRequest(http.MethodPost, "connectorID", connID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should handle request with body content", func(ctx SpecContext) {
			expectedRedirectURL := "https://example.com/redirect"
			bodyContent := []byte(`{"key": "value"}`)

			m.EXPECT().PaymentServiceUsersCompleteLinkFlow(
				gomock.Any(),
				connID,
				models.HTTPCallInformation{
					QueryValues: url.Values{},
					Headers:     http.Header{},
					Body:        bodyContent,
				},
			).Return(expectedRedirectURL, nil)

			req := prepareQueryRequestWithBody(http.MethodPost, strings.NewReader(`{"key": "value"}`), "connectorID", connID.String())

			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusTemporaryRedirect, "")
			Expect(w.Header().Get("Location")).To(Equal(expectedRedirectURL))
		})
	})
})

// errorReader is a mock reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func (e *errorReader) Close() error {
	return nil
}
