package httpwrapper_test

import (
	"context"
	"fmt"
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
		DescribeTable("classifies retryable 4xx codes",
			func(ctx SpecContext, statusCode int, expected error) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?code=%d", server.URL, statusCode), http.NoBody)
				Expect(err).To(BeNil())

				res := &errorRes{}
				code, doErr := client.Do(context.Background(), req, &successRes{}, res)
				Expect(code).To(Equal(statusCode))
				Expect(doErr).To(MatchError(expected))
				Expect(res.Code).To(Equal("err123"))
			},
			Entry("408 Request Timeout", http.StatusRequestTimeout, httpwrapper.ErrStatusCodeRequestTimeout),
			Entry("421 Misdirected Request", http.StatusMisdirectedRequest, httpwrapper.ErrStatusCodeMisdirectedRequest),
			Entry("423 Locked", http.StatusLocked, httpwrapper.ErrStatusCodeLocked),
			Entry("425 Too Early", http.StatusTooEarly, httpwrapper.ErrStatusCodeTooEarly),
			Entry("429 Too Many Requests", http.StatusTooManyRequests, httpwrapper.ErrStatusCodeTooManyRequests),
		)
		It("responds with error when HTTP request fails", func(ctx SpecContext) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "notaurl", http.NoBody)
			Expect(err).To(BeNil())

			res := &errorRes{}
			code, doErr := client.Do(context.Background(), req, &successRes{}, res)
			Expect(code).To(Equal(0))
			Expect(doErr).To(MatchError(ContainSubstring("failed to make request")))
		})
	})
})
