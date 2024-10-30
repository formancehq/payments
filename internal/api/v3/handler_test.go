package v3

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV3Handlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API v3 Suite")
}

func assertExpectedResponse(res *http.Response, expectedStatusCode int, expectedBodyString string) {
	defer res.Body.Close()
	Expect(res.StatusCode).To(Equal(expectedStatusCode))

	data, err := ioutil.ReadAll(res.Body)
	Expect(err).To(BeNil())
	Expect(data).To(ContainSubstring(expectedBodyString))
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
	return prepareQueryRequestWithBody(method, key, val, body)
}

func prepareQueryRequest(key string, val string) *http.Request {
	return prepareQueryRequestWithBody(http.MethodGet, key, val, nil)
}

func prepareQueryRequestWithBody(method string, key string, val string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, "/", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
