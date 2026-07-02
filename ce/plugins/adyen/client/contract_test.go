//go:build contract

// Package client contract test for the Adyen connector.
//
// This is a CONTRACT test: it calls the real Adyen test environment over the
// network through the same client.Client the connector uses, and asserts that
// the responses the Payments project depends on have not drifted in schema
// (field presence + types) or in list ordering. It is gated behind the
// `contract` build tag so it never runs as part of `just tests` (which only
// enables `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	ADYEN_CONTRACT_API_KEY=... ADYEN_CONTRACT_COMPANY_ID=... \
//	    just contract-tests adyen
//
// Without the two env vars the suite Skips rather than fails, so it is safe to
// run anywhere.
package client

import (
	"context"
	"os"
	"regexp"
	"testing"

	"github.com/formancehq/payments/pkg/domain/contracttest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAdyenContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Adyen Contract Suite")
}

const (
	// Contract Webhook ID is the webhook ID registered in the Adyen platform.
	contractWebhookID = "WBHK4296F22322BR5PLFZTN2M82GJ3"

	// contractWebhookURL is a syntactically valid public URL used only to
	// create the temporary webhook. NOTE: real end-to-end webhook *delivery*
	// (Adyen POSTing a notification to this URL and us verifying its HMAC) is
	// NOT exercised here — CI has no public ingress. Delivery/HMAC verification
	// of a real payload is covered by the unit tests with recorded payloads.
	// This contract test only validates the management-API representation of
	// the webhook (creation, retrievable shape, deletion).
	// We picked this URL only because it was already defined in Adyen console.
	contractWebhookURL = "https://olwwbvmextkh-tyzr.paulnicolas.formance.dev/api/payments/v3/connectors/webhooks/eyJQcm92aWRlciI6ImFkeWVuIiwiUmVmZXJlbmNlIjoiMzU1MDFiNTQtNGZlYy00YTcwLWIxMDMtZDgxZjI1NzAzMmYyIn0/standard"
)

// adyenHMACKeyPattern matches Adyen's generated HMAC keys (hex string).
var adyenHMACKeyPattern = regexp.MustCompile(`^[0-9A-Fa-f]{32,}$`)

// expectedMerchantIDs pins the known, seeded order of merchant accounts in the
// Adyen test company used for contract testing, in the order they are expected
// to be returned.
var expectedMerchantIDs = []string{
	"Formance814ECOM",
	"Formance814_AAAAAA_TEST",
	"Formance814_Formance1_TEST",
	"Formance814_USER_1234_TEST",
	"Formance814_USER_2345_TEST",
	"Formance814_USER_3456_TEST",
	"Formance814_USER_4567_TEST",
}

var _ = Describe("Adyen API contract", func() {
	var (
		ctx       context.Context
		c         Client
		companyID string
	)

	BeforeEach(func() {
		apiKey := os.Getenv("ADYEN_CONTRACT_API_KEY")
		companyID = os.Getenv("ADYEN_CONTRACT_COMPANY_ID")
		if apiKey == "" || companyID == "" {
			Skip("ADYEN_CONTRACT_API_KEY and ADYEN_CONTRACT_COMPANY_ID must be set to run the Adyen contract test")
		}

		ctx = context.Background()
		// liveEndpointPrefix empty => common.TestEnv (Adyen test environment).
		c = New("adyen", apiKey, "", "", companyID, "")
	})

	Describe("GetMerchantAccounts", func() {
		It("returns merchant accounts whose shape matches what the connector consumes", func() {
			merchants, err := c.GetMerchantAccounts(ctx, 1, 100)
			Expect(err).To(BeNil())
			Expect(merchants).ToNot(BeEmpty())

			ids := make([]string, 0, len(merchants))
			for _, m := range merchants {
				// Fields the connector relies on must be present and typed.
				Expect(m.Id).ToNot(BeNil())
				Expect(*m.Id).ToNot(BeEmpty())
				Expect(m.Name).ToNot(BeNil())
				Expect(m.Status).ToNot(BeNil())
				ids = append(ids, *m.Id)
			}

			if contracttest.BootstrapEnabled("ADYEN") {
				contracttest.LogBootstrap("expectedMerchantIDs", ids)
			}
		})

		It("returns merchant accounts in the expected, stable order", func() {
			if len(expectedMerchantIDs) == 0 {
				Skip("expectedMerchantIDs is not populated — fill in the seeded test " +
					"company's merchant account IDs (in order) to enable the ordering contract")
			}

			merchants, err := c.GetMerchantAccounts(ctx, 1, 100)
			Expect(err).To(BeNil())

			gotIDs := make([]string, 0, len(merchants))
			for _, m := range merchants {
				Expect(m.Id).ToNot(BeNil())
				gotIDs = append(gotIDs, *m.Id)
			}

			// Single call, exact in-order comparison against the pinned list.
			Expect(gotIDs).To(Equal(expectedMerchantIDs))
		})
	})

	Describe("webhook lifecycle", func() {
		// cc gives access to the unexported helpers (searchWebhook /
		// standardWebhook) so we can re-fetch and validate the created webhook
		// at the management-API contract level. The test lives in package
		// client precisely to enable this.
		var cc *client

		BeforeEach(func() {
			cc = c.(*client)
			// Delete-before-create: clear any webhook left over from a prior
			// failed run so the create path actually runs and returns a fresh
			// HMAC key.
			Expect(cc.DeleteWebhook(ctx, contractWebhookID)).To(BeNil())
			cc.standardWebhook = nil
		})

		It("creates a webhook with a well-formed HMAC key and a valid, retrievable shape", func() {
			DeferCleanup(func() {
				cc.standardWebhook = nil
				Expect(cc.DeleteWebhook(ctx, contractWebhookID)).To(BeNil())
			})

			resp, err := cc.CreateWebhook(ctx, contractWebhookURL, contractWebhookID)
			Expect(err).To(BeNil())
			Expect(resp.HMACKey).ToNot(BeEmpty())
			Expect(adyenHMACKeyPattern.MatchString(resp.HMACKey)).To(BeTrue(),
				"HMAC key %q does not look like an Adyen HMAC key", resp.HMACKey)

			// Re-fetch the webhook from the API to validate the representation
			// that comes back is valid (not just that creation returned 200).
			cc.standardWebhook = nil
			Expect(cc.searchWebhook(ctx, contractWebhookID)).To(BeNil())
			Expect(cc.standardWebhook).ToNot(BeNil())

			wh := cc.standardWebhook
			Expect(wh.Id).ToNot(BeNil())
			Expect(*wh.Id).ToNot(BeEmpty())
			Expect(wh.Type).To(Equal("standard"))
			Expect(wh.Active).To(BeTrue())
			Expect(wh.Url).To(Equal(contractWebhookURL))
			Expect(wh.CommunicationFormat).To(Equal("json"))
			Expect(wh.Description).ToNot(BeNil())
			Expect(*wh.Description).To(Equal(contractWebhookID))
		})
	})
})
