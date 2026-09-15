//go:build !contract

package client

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This test lives in package client (not client_test) so it can reach into
// the unexported *client to assert on the wrapped plaid.APIClient's
// configured server, matching the pattern already used by
// formance_open_banking_post_test.go.
var _ = Describe("New with a BaseURL override", func() {
	var (
		requests []*http.Request
		server   *httptest.Server
	)

	BeforeEach(func() {
		requests = nil
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Clone(r.Context()))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"fake-access-token","item_id":"fake-item-id","request_id":"fake-request-id"}`))
		}))
		DeferCleanup(server.Close)
	})

	It("routes plaid-go SDK calls to the overridden base URL instead of Production/Sandbox", func(ctx SpecContext) {
		c, err := New("plaid", "client-id", "client-secret", "conn-1", false, server.URL)
		Expect(err).To(BeNil())

		resp, err := c.ExchangePublicToken(ctx, ExchangePublicTokenRequest{PublicToken: "public-token"})
		Expect(err).To(BeNil())
		Expect(resp.AccessToken).To(Equal("fake-access-token"))
		Expect(resp.ItemID).To(Equal("fake-item-id"))

		Expect(requests).To(HaveLen(1))
		Expect(requests[0].Host).To(Equal(mustHost(server.URL)))
	})

	It("ignores IsSandbox when BaseURL is set", func(ctx SpecContext) {
		c, err := New("plaid", "client-id", "client-secret", "conn-1", true, server.URL)
		Expect(err).To(BeNil())

		_, err = c.ExchangePublicToken(ctx, ExchangePublicTokenRequest{PublicToken: "public-token"})
		Expect(err).To(BeNil())
		Expect(requests).To(HaveLen(1))
		Expect(requests[0].Host).To(Equal(mustHost(server.URL)))
	})
})

func mustHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u.Host
}
