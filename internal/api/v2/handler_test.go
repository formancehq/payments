package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV2Handlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API v2 Suite")
}

func assertExpectedResponse(res *http.Response, expectedStatusCode int, expectedBodyString string) {
	defer res.Body.Close()
	Expect(res.StatusCode).To(Equal(expectedStatusCode))

	data, err := io.ReadAll(res.Body)
	Expect(err).To(BeNil())
	Expect(string(data)).To(ContainSubstring(expectedBodyString))
}

func prepareJSONRequest(method string, a any) *http.Request {
	b, err := json.Marshal(a)
	Expect(err).To(BeNil())
	body := bytes.NewReader(b)
	return httptest.NewRequest(method, "/", body)
}

func prepareJSONRequestWithQuery(method string, key string, val string, a any) *http.Request {
	b, err := json.Marshal(a)
	Expect(err).To(BeNil())
	body := bytes.NewReader(b)
	return prepareQueryRequestWithBody(method, body, key, val)
}

func prepareQueryRequest(method string, args ...string) *http.Request {
	return prepareQueryRequestWithBody(method, nil, args...)
}

func prepareQueryRequestWithBody(method string, body io.Reader, args ...string) *http.Request {
	req := httptest.NewRequest(method, "/", body)
	rctx := chi.NewRouteContext()
	appendToRouteContext(rctx, args...)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func prepareQueryRequestWithPath(path string, args ...string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rctx := chi.NewRouteContext()
	appendToRouteContext(rctx, args...)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func appendToRouteContext(rctx *chi.Context, args ...string) {
	if len(args)%2 != 0 {
		log.Fatalf("arguments must be provided in key value pairs: %s", args)
	}
	for i := 0; i < len(args); i += 2 {
		val := args[i+1]
		rctx.URLParams.Add(args[i], val)
	}
}
