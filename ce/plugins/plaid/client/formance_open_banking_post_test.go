//go:build !contract

package client

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/pkg/domain/httpwrapper"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// newRedirectTestClient builds a *client with only the fields
// FormanceOpenBankingRedirect touches populated - this test lives in package
// client (not client_test) specifically to construct these directly, rather
// than going through the exported New (which also sets up a real Plaid SDK
// client this test never exercises).
func newRedirectTestClient(formanceStackEndpoint, connectorID string) *client {
	return &client{
		formanceHTTPClient:    httpwrapper.NewClient(&httpwrapper.Config{}),
		formanceStackEndpoint: formanceStackEndpoint,
		connectorID:           connectorID,
	}
}

var _ = Describe("FormanceOpenBankingRedirect", func() {
	var (
		requests []*http.Request
		server   *httptest.Server
	)

	BeforeEach(func() {
		requests = nil
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Clone(r.Context()))
			w.WriteHeader(http.StatusOK)
		}))
		DeferCleanup(server.Close)
	})

	It("posts to the caller-supplied RedirectURL with the expected query parameters", func(ctx SpecContext) {
		c := newRedirectTestClient(server.URL+"/api/payments/v3", "unused-connector-id")
		attemptID := uuid.New()

		err := c.FormanceOpenBankingRedirect(ctx, FormanceOpenBankingRedirectRequest{
			RedirectURL: server.URL + "/custom/redirect",
			LinkToken:   "lt-1",
			PublicToken: "pt-1",
			AttemptID:   attemptID,
		})
		Expect(err).To(BeNil())
		Expect(requests).To(HaveLen(1))

		req := requests[0]
		Expect(req.Method).To(Equal(http.MethodPost))
		Expect(req.URL.Path).To(Equal("/custom/redirect"))
		Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))

		q := req.URL.Query()
		Expect(q.Get(LinkTokenQueryParamID)).To(Equal("lt-1"))
		Expect(q.Get(PublicTokenQueryParamID)).To(Equal("pt-1"))
		Expect(q.Get(models.NoRedirectQueryParamID)).To(Equal("true"))
		Expect(q.Get(StateQueryParamID)).To(Equal(models.CallbackState{AttemptID: attemptID}.String()))
	})

	It("preserves an existing query string on the caller-supplied RedirectURL", func(ctx SpecContext) {
		c := newRedirectTestClient(server.URL+"/api/payments/v3", "unused-connector-id")

		err := c.FormanceOpenBankingRedirect(ctx, FormanceOpenBankingRedirectRequest{
			RedirectURL: server.URL + "/custom/redirect?attemptID=abc",
			LinkToken:   "lt-1",
			PublicToken: "pt-1",
		})
		Expect(err).To(BeNil())
		Expect(requests).To(HaveLen(1))
		Expect(requests[0].URL.Query().Get("attemptID")).To(Equal("abc"))
	})

	It("falls back to the connectorID/formanceStackEndpoint construction when RedirectURL is empty", func(ctx SpecContext) {
		c := newRedirectTestClient(server.URL+"/api/payments/v3", "conn-xyz")
		attemptID := uuid.New()

		err := c.FormanceOpenBankingRedirect(ctx, FormanceOpenBankingRedirectRequest{
			RedirectURL: "",
			LinkToken:   "lt-2",
			PublicToken: "pt-2",
			AttemptID:   attemptID,
		})
		Expect(err).To(BeNil())
		Expect(requests).To(HaveLen(1))
		Expect(requests[0].URL.Path).To(Equal("/api/payments/v3/connectors/open-banking/conn-xyz/redirect"))
		Expect(requests[0].URL.Query().Get(LinkTokenQueryParamID)).To(Equal("lt-2"))
	})

	It("returns an error for a malformed RedirectURL, without making any HTTP call", func(ctx SpecContext) {
		c := newRedirectTestClient(server.URL+"/api/payments/v3", "conn-xyz")

		err := c.FormanceOpenBankingRedirect(ctx, FormanceOpenBankingRedirectRequest{
			RedirectURL: "not a url\x7f",
		})
		Expect(err).ToNot(BeNil())
		Expect(requests).To(BeEmpty())
	})

	It("returns an error when the fallback construction's stack endpoint is malformed", func(ctx SpecContext) {
		c := newRedirectTestClient("not a url\x7f", "conn-xyz")

		err := c.FormanceOpenBankingRedirect(ctx, FormanceOpenBankingRedirectRequest{
			RedirectURL: "",
		})
		Expect(err).ToNot(BeNil())
		Expect(requests).To(BeEmpty())
	})

	It("propagates an HTTP server error from the destination", func(ctx SpecContext) {
		errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		DeferCleanup(errServer.Close)

		c := newRedirectTestClient(errServer.URL, "conn-xyz")

		err := c.FormanceOpenBankingRedirect(ctx, FormanceOpenBankingRedirectRequest{
			RedirectURL: errServer.URL + "/redirect",
		})
		Expect(err).ToNot(BeNil())
		Expect(errors.Is(err, httpwrapper.ErrStatusCodeServerError)).To(BeTrue())
	})
})
