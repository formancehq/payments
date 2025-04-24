package httpwrapper_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

type successRes struct {
	ID string `json:"id"`
}

type errorRes struct {
	Code string `json:"code"`
}

var _ = Describe("ClientWrapper", func() {
	var (
		config *httpwrapper.Config
		client httpwrapper.Client
		server *httptest.Server
	)

	BeforeEach(func() {
		config = &httpwrapper.Config{Timeout: 30 * time.Millisecond}
		client = httpwrapper.NewClient(config)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			params, err := url.ParseQuery(r.URL.RawQuery)
			Expect(err).To(BeNil())

			code := params.Get("code")
			statusCode, err := strconv.Atoi(code)
			Expect(err).To(BeNil())
			if statusCode == http.StatusOK {
				_, err := w.Write([]byte(`{"id":"someid"}`))
				Expect(err).To(BeNil())
				return
			}

			w.WriteHeader(statusCode)
			_, err = w.Write([]byte(`{"code":"err123"}`))
			Expect(err).To(BeNil())
		}))
	})
	AfterEach(func() {
		server.Close()
	})

	Context("making a request with default client settings", func() {
		It("unmarshals successful responses when acceptable status code seen", func(ctx SpecContext) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"?code=200", http.NoBody)
			Expect(err).To(BeNil())

			res := &successRes{}
			code, doErr := client.Do(context.Background(), req, res, nil)
			Expect(code).To(Equal(http.StatusOK))
			Expect(doErr).To(BeNil())
			Expect(res.ID).To(Equal("someid"))
		})
		It("unmarshals error responses when bad status code seen", func(ctx SpecContext) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"?code=500", http.NoBody)
			Expect(err).To(BeNil())

			res := &errorRes{}
			code, doErr := client.Do(context.Background(), req, &successRes{}, res)
			Expect(code).To(Equal(http.StatusInternalServerError))
			Expect(doErr).To(MatchError(httpwrapper.ErrStatusCodeServerError))
			Expect(res.Code).To(Equal("err123"))
		})
		It("unmarshals error responses when http client error seen", func(ctx SpecContext) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"?code=400", http.NoBody)
			Expect(err).To(BeNil())

			res := &errorRes{}
			code, doErr := client.Do(context.Background(), req, &successRes{}, res)
			Expect(code).To(Equal(http.StatusBadRequest))
			Expect(doErr).To(MatchError(httpwrapper.ErrStatusCodeClientError))
			Expect(res.Code).To(Equal("err123"))
		})
		It("responds with error when HTTP request fails", func(ctx SpecContext) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "notaurl", http.NoBody)
			Expect(err).To(BeNil())

			res := &errorRes{}
			code, doErr := client.Do(context.Background(), req, &successRes{}, res)
			Expect(code).To(Equal(0))
			Expect(doErr).To(MatchError(ContainSubstring("failed to make request")))
		})
	})
	
	Context("handling different error scenarios", func() {
		It("handles context cancellation", func(ctx SpecContext) {
			cancelCtx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel the context immediately
			
			req, err := http.NewRequestWithContext(cancelCtx, http.MethodGet, server.URL+"?code=200", http.NoBody)
			Expect(err).To(BeNil())
			
			res := &successRes{}
			code, doErr := client.Do(cancelCtx, req, res, nil)
			Expect(code).To(Equal(0))
			Expect(doErr).To(MatchError(ContainSubstring("context canceled")))
		})
		
		It("handles timeout errors", func(ctx SpecContext) {
			timeoutConfig := &httpwrapper.Config{Timeout: 1 * time.Nanosecond}
			timeoutClient := httpwrapper.NewClient(timeoutConfig)
			
			slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			}))
			defer slowServer.Close()
			
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, slowServer.URL, http.NoBody)
			Expect(err).To(BeNil())
			
			res := &successRes{}
			code, doErr := timeoutClient.Do(context.Background(), req, res, nil)
			Expect(code).To(Equal(0))
			Expect(doErr).To(MatchError(ContainSubstring("timeout")))
		})
		
		It("handles too many requests status code", func(ctx SpecContext) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"?code=429", http.NoBody)
			Expect(err).To(BeNil())
			
			res := &errorRes{}
			code, doErr := client.Do(context.Background(), req, &successRes{}, res)
			Expect(code).To(Equal(http.StatusTooManyRequests))
			Expect(doErr).To(MatchError(httpwrapper.ErrStatusCodeTooManyRequests))
			Expect(res.Code).To(Equal("err123"))
		})
	})
})
