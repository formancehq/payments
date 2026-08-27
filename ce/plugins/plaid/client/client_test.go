//go:build !contract

package client

import (
	"github.com/plaid/plaid-go/v34/plaid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This test lives in package client (not client_test) so it can reach into
// the unexported *client to assert on the wrapped plaid.APIClient's
// configured server, matching the pattern used by base_url_test.go.
var _ = Describe("New without a BaseURL override", func() {
	It("defaults to the Plaid Production environment", func() {
		c, err := New("plaid", "client-id", "client-secret", "conn-1", false, "")
		Expect(err).To(BeNil())

		impl, ok := c.(*client)
		Expect(ok).To(BeTrue())
		Expect(impl.client.GetConfig().Servers).To(HaveLen(1))
		Expect(impl.client.GetConfig().Servers[0].URL).To(Equal(string(plaid.Production)))
	})

	It("uses the Plaid Sandbox environment when IsSandbox is true", func() {
		c, err := New("plaid", "client-id", "client-secret", "conn-1", true, "")
		Expect(err).To(BeNil())

		impl, ok := c.(*client)
		Expect(ok).To(BeTrue())
		Expect(impl.client.GetConfig().Servers).To(HaveLen(1))
		Expect(impl.client.GetConfig().Servers[0].URL).To(Equal(string(plaid.Sandbox)))
	})
})
