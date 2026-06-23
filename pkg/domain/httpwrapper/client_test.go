package httpwrapper_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/httpwrapper"
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

// trackingBody records whether it was read to EOF (drained) and closed.
type trackingBody struct {
	reader  *bytes.Reader
	drained bool
	closed  bool
}

func (b *trackingBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if errors.Is(err, io.EOF) {
		b.drained = true
	}
	return n, err
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

// trackingTransport returns a fixed response whose body tracks read/close state.
type trackingTransport struct {
	statusCode int
	body       *trackingBody
}

func (t *trackingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.statusCode,
		Body:       t.body,
		Header:     make(http.Header),
	}, nil
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
		DescribeTable("classifies url.Error transport failures as client errors",
			func(ctx SpecContext, rawURL string) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
				Expect(err).To(BeNil())

				code, doErr := client.Do(context.Background(), req, nil, nil)
				Expect(code).To(Equal(0))
				Expect(errors.Is(doErr, httpwrapper.ErrStatusCodeClientError)).To(BeTrue())
			},
			Entry("no such host", "http://this-host-does-not-exist.invalid"),
			Entry("unsupported protocol scheme", "ftp://example.com"),
			Entry("relative URL with no scheme", "notaurl"),
		)
		It("classifies http client timeout as a request timeout error", func() {
			slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
			}))
			defer slowServer.Close()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, slowServer.URL, http.NoBody)
			Expect(err).To(BeNil())

			code, doErr := client.Do(context.Background(), req, nil, nil)
			Expect(code).To(Equal(0))
			Expect(errors.Is(doErr, httpwrapper.ErrStatusCodeRequestTimeout)).To(BeTrue())
			Expect(errors.Is(doErr, httpwrapper.ErrStatusCodeClientError)).To(BeFalse())
		})
		It("classifies context deadline exceeded as a request timeout error", func() {
			slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
			}))
			defer slowServer.Close()

			deadlineCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			req, err := http.NewRequestWithContext(deadlineCtx, http.MethodGet, slowServer.URL, http.NoBody)
			Expect(err).To(BeNil())

			code, doErr := client.Do(deadlineCtx, req, nil, nil)
			Expect(code).To(Equal(0))
			Expect(errors.Is(doErr, httpwrapper.ErrStatusCodeRequestTimeout)).To(BeTrue())
			Expect(errors.Is(doErr, httpwrapper.ErrStatusCodeClientError)).To(BeFalse())
		})
		It("does not classify context cancellation as a client error", func() {
			cancelCtx, cancel := context.WithCancel(context.Background())
			cancel()

			req, err := http.NewRequestWithContext(cancelCtx, http.MethodGet, server.URL+"?code=200", http.NoBody)
			Expect(err).To(BeNil())

			code, doErr := client.Do(cancelCtx, req, nil, nil)
			Expect(code).To(Equal(0))
			Expect(errors.Is(doErr, httpwrapper.ErrStatusCodeClientError)).To(BeFalse())
			Expect(doErr).To(MatchError(ContainSubstring("failed to make request")))
		})
	})

	Context("draining and closing the response body", func() {
		DescribeTable("always drains and closes the body, including on early-return paths",
			func(ctx SpecContext, statusCode int, expectedBody, errorBody any) {
				body := &trackingBody{reader: bytes.NewReader([]byte(`{"id":"someid"}`))}
				trackingClient := httpwrapper.NewClient(&httpwrapper.Config{
					Transport: &trackingTransport{statusCode: statusCode, body: body},
				})

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", http.NoBody)
				Expect(err).To(BeNil())

				code, _ := trackingClient.Do(context.Background(), req, expectedBody, errorBody)
				Expect(code).To(Equal(statusCode))
				Expect(body.drained).To(BeTrue(), "response body should be drained for keep-alive reuse")
				Expect(body.closed).To(BeTrue(), "response body should be closed")
			},
			Entry("success with nil expectedBody (early return)", http.StatusOK, nil, nil),
			Entry("error status with nil errorBody (early return)", http.StatusInternalServerError, nil, nil),
			Entry("success with expectedBody (read path)", http.StatusOK, &successRes{}, nil),
			Entry("error status with errorBody (read path)", http.StatusInternalServerError, &successRes{}, &errorRes{}),
		)
	})
})
