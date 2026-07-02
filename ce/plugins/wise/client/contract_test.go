//go:build contract

// Package client contract test for the Wise connector.
//
// This is a CONTRACT test: it calls the real Wise sandbox over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types) or in list ordering. It is gated behind the `contract`
// build tag so it never runs as part of `just tests` (which only enables
// `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	WISE_CONTRACT_API_KEY=... just contract-tests wise
//
// Sandbox and production are DIFFERENT hosts (prod api.wise.com is hardcoded
// in client.go; the sandbox lives on api.sandbox.transferwise.tech), so the
// sandbox host is hardcoded here (contractEndpoint) and only the API key is
// injected — a sandbox token can never hit production. Without
// WISE_CONTRACT_API_KEY the suite Skips rather than fails, so it is safe to
// run anywhere.
//
// Ordering: every Wise list the connector consumes is watermarked on the
// immutable numeric ID assuming ascending order (profiles.go, accounts.go,
// external_accounts.go — which even requests sort=id,asc — and payments.go),
// so the ordering contract is a monotonic non-decreasing-ID assertion per
// list. No pinned IDs, no bootstrap step.
//
// Money movement: the CreateQuote/CreateTransfer specs create an UNFUNDED
// transfer — the connector (and this suite) never calls Wise's
// transfer-funding endpoint, so no money can move; Wise auto-expires unfunded
// transfers, so per-run accumulation is self-limiting. customerTransactionId
// must be a UUID, unique per run (contracttest.UUIDRef). CreatePayout is not
// exercised: its request/response wire contract is byte-identical to
// CreateTransfer (same POST v1/transfers, structurally identical types).
// GetPayout is excluded entirely — no ingestion path calls it, and it hits
// the same v1/transfers/{id} endpoint as GetTransfer.
package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/formancehq/payments/pkg/domain/contracttest"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWiseContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wise Contract Suite")
}

// contractPageSize mirrors the connector's PAGE_SIZE (config.go: Wise's max
// page size is 100).
const contractPageSize = 100

// contractWalkPageSize is deliberately small so the full-list walkers cross
// page boundaries against the modest seeded sandbox (~12 recipients, ~16
// transfers). The paging CONTRACT — seekPosition semantics, offset/limit
// consistency, stable ordering across pages — is what the connector's
// watermarks depend on, and a single-page walk would never exercise it.
const contractWalkPageSize = 5

// contractEndpoint is the Wise sandbox host. Sandbox and production are
// different hosts, so this is hardcoded and only the API key is a secret.
const contractEndpoint = "https://api.sandbox.transferwise.tech"

// contractWebhookURL must be a live HTTPS endpoint that answers POSTs with a
// 2xx: Wise VALIDATES the callback at subscription-create time by POSTing a
// test notification (the X-Test-Notification event the connector's
// VerifyWebhook special-cases) and rejects the create with 422
// INVALID_CALLBACK_URL ("Callback endpoint did not respond successfully")
// when the receiver does not answer — so a dead placeholder URL cannot be
// used, unlike Adyen/Column. postman-echo answers 200 to any POST. Real
// end-to-end *delivery* (signature verification of a real event) is still
// NOT exercised — the echo endpoint only satisfies the create-time probe;
// delivery/signature verification is covered by the unit tests. The
// lifecycle spec self-probes this URL first and Skips when it is down, so
// echo-service downtime reads as a Skip, not as upstream drift.
const contractWebhookURL = "https://postman-echo.com/post"

// collectAllRecipientIDs pages through every recipient account the same way
// fetchExternalAccounts does (seekPosition walk; the client requests
// sort=id,asc) and returns the IDs in list order. The loop cap is an
// infinite-loop guard, not a call count — the walk stops when the API reports
// a short page.
func collectAllRecipientIDs(ctx context.Context, c Client, profileID uint64) ([]uint64, error) {
	var ids []uint64
	var seek uint64
	for i := 0; i < 5000; i++ {
		page, err := c.GetRecipientAccounts(ctx, profileID, contractWalkPageSize, seek)
		if err != nil {
			return nil, err
		}
		for _, ra := range page.Content {
			ids = append(ids, ra.ID)
		}
		if len(page.Content) < contractWalkPageSize || page.SeekPositionForNext == 0 {
			break
		}
		seek = page.SeekPositionForNext
	}
	return ids, nil
}

// collectAllTransfers walks the transfer list the same way fetchNextPayments
// does (offset pages, offset advancing by the page size) and returns every
// transfer in list order. The loop cap is an infinite-loop guard.
func collectAllTransfers(ctx context.Context, c Client, profileID uint64) ([]Transfer, error) {
	var transfers []Transfer
	offset := 0
	for i := 0; i < 5000; i++ {
		// The offset must stay a multiple of the limit (Wise rejects
		// "inconsistent pagination" otherwise — see payments.go), which
		// advancing by the page size preserves.
		page, err := c.GetTransfers(ctx, profileID, offset, contractWalkPageSize)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, page...)
		if len(page) < contractWalkPageSize {
			break
		}
		offset += contractWalkPageSize
	}
	return transfers, nil
}

var _ = Describe("Wise API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		apiKey := os.Getenv("WISE_CONTRACT_API_KEY")
		if apiKey == "" {
			Skip("WISE_CONTRACT_API_KEY must be set to run the Wise contract test")
		}

		ctx = context.Background()
		c = newWithEndpoint("wise", apiKey, contractEndpoint)
	})

	// contractProfile returns the profile every other read hangs off (the
	// connector's workflow fans out from fetch_profiles). Prefer the business
	// profile — that is where sandbox balances/recipients/transfers live.
	contractProfile := func() Profile {
		profiles, err := c.GetProfiles(ctx)
		Expect(err).To(BeNil())
		Expect(profiles).ToNot(BeEmpty())
		for _, p := range profiles {
			if strings.EqualFold(p.Type, "business") {
				return p
			}
		}
		return profiles[0]
	}

	Describe("GetProfiles", func() {
		It("returns profiles whose shape and ordering match what the connector consumes", func() {
			profiles, err := c.GetProfiles(ctx)
			Expect(err).To(BeNil())
			Expect(profiles).ToNot(BeEmpty())

			ids := make([]uint64, 0, len(profiles))
			for _, p := range profiles {
				// id is the fetch watermark (profilesState.LastProfileID) and
				// the FromPayload scope for every other fetch.
				Expect(p.ID).ToNot(BeZero())
				ids = append(ids, p.ID)
			}

			// fetchNextProfiles skips profile.ID <= LastProfileID, assuming
			// the list is ordered by ascending ID.
			contracttest.AssertNonDecreasing(ids, "profiles ids")
		})
	})

	Describe("GetBalances", func() {
		It("returns balances (our accounts) whose shape and ordering match what the connector consumes", func() {
			profile := contractProfile()

			balances, err := c.GetBalances(ctx, profile.ID)
			Expect(err).To(BeNil())
			if len(balances) == 0 {
				Skip("the sandbox profile has no balances — open one via the API (see the seeding section of .omc/specs/contract-tests-wise.md) to enable the balance specs")
			}

			ids := make([]uint64, 0, len(balances))
			for _, b := range balances {
				// id is the account Reference and the fetch watermark
				// (accountsState.LastAccountID).
				Expect(b.ID).ToNot(BeZero())
				// creationTime is PSPAccount.CreatedAt; a non-RFC3339 value
				// would already have failed the time.Time unmarshal above, an
				// omitted one would silently ingest as the zero time.
				Expect(b.CreationTime.IsZero()).To(BeFalse(), "balance %d creationTime is missing", b.ID)
				// amount.currency feeds DefaultAsset. Name is soft (address
				// taken, empty tolerated) — not asserted.
				Expect(b.Amount.Currency).ToNot(BeEmpty())
				contracttest.AssertDecimalAmount(b.Amount.Value, "balance amount.value")
				ids = append(ids, b.ID)
			}

			// fetchNextAccounts skips balance.ID <= LastAccountID, assuming
			// the list is ordered by ascending ID.
			contracttest.AssertNonDecreasing(ids, "balances ids")
		})
	})

	Describe("GetBalance", func() {
		It("returns a balance by ID whose shape matches what the connector consumes", func() {
			profile := contractProfile()

			balances, err := c.GetBalances(ctx, profile.ID)
			Expect(err).To(BeNil())
			if len(balances) == 0 {
				Skip("the sandbox profile has no balances — open one via the API (see the seeding section of .omc/specs/contract-tests-wise.md) to enable the balance specs")
			}

			balance, err := c.GetBalance(ctx, profile.ID, balances[0].ID)
			Expect(err).To(BeNil())
			Expect(balance).ToNot(BeNil())

			Expect(balance.ID).To(Equal(balances[0].ID))
			// fetchNextBalances hard-parses amount.value as a decimal string
			// and errors on an unsupported/absent currency.
			Expect(balance.Amount.Currency).ToNot(BeEmpty())
			contracttest.AssertDecimalAmount(balance.Amount.Value, "balance amount.value")
			// modificationTime is PSPBalance.CreatedAt.
			Expect(balance.ModificationTime.IsZero()).To(BeFalse(), "balance modificationTime is missing")
		})
	})

	Describe("GetRecipientAccounts", func() {
		It("returns recipient accounts whose shape and ordering match what the connector consumes", func() {
			profile := contractProfile()

			page, err := c.GetRecipientAccounts(ctx, profile.ID, contractPageSize, 0)
			Expect(err).To(BeNil())
			if len(page.Content) == 0 {
				Skip("the sandbox profile has no recipient accounts — seed one via the sandbox UI to enable the recipient/transfer specs")
			}

			for _, ra := range page.Content {
				// id is the external-account Reference and the fetch watermark
				// (externalAccountsState.LastSeekPosition). currency feeds
				// DefaultAsset. name.fullName is soft (address taken, empty
				// tolerated) — not asserted.
				Expect(ra.ID).ToNot(BeZero())
				Expect(ra.Currency).ToNot(BeEmpty())
			}

			// fillExternalAccounts skips id <= LastSeekPosition and the client
			// requests sort=id,asc: the walk (which also exercises the
			// seekPositionForNext pagination cursor) must yield ascending IDs.
			ids, err := collectAllRecipientIDs(ctx, c, profile.ID)
			Expect(err).To(BeNil())
			contracttest.AssertNonDecreasing(ids, "recipient account ids")
		})
	})

	Describe("GetRecipientAccount", func() {
		It("returns a recipient account by ID whose shape matches what the connector consumes", func() {
			profile := contractProfile()

			page, err := c.GetRecipientAccounts(ctx, profile.ID, contractPageSize, 0)
			Expect(err).To(BeNil())
			if len(page.Content) == 0 {
				Skip("the sandbox profile has no recipient accounts — seed one via the sandbox UI to enable the recipient/transfer specs")
			}

			ra, err := c.GetRecipientAccount(ctx, page.Content[0].ID)
			Expect(err).To(BeNil())
			Expect(ra).ToNot(BeNil())

			Expect(ra.ID).To(Equal(page.Content[0].ID))
			// GetTransfers dereferences .Profile to route the balance
			// enrichment of each transfer's source/target.
			Expect(ra.Profile).ToNot(BeZero())
		})
	})

	Describe("GetTransfers", func() {
		It("returns transfers whose shape and ordering match what the connector consumes", func() {
			profile := contractProfile()

			transfers, err := collectAllTransfers(ctx, c, profile.ID)
			// A nil error already proves every transfer's `created` parsed as
			// "2006-01-02 15:04:05" — UnmarshalJSON errors otherwise.
			Expect(err).To(BeNil())
			// The seeded sandbox always has transfers (and each run adds one),
			// so an empty list means the endpoint or its filters drifted.
			Expect(transfers).ToNot(BeEmpty())

			ids := make([]uint64, 0, len(transfers))
			for _, tr := range transfers {
				// id is the payment Reference and the fetch watermark
				// (paymentsState.LastTransferID).
				Expect(tr.ID).ToNot(BeZero())
				Expect(tr.CreatedAt.IsZero()).To(BeFalse(), "transfer %d created is missing", tr.ID)
				// targetValue/targetCurrency are hard-parsed by
				// fromTransferToPayment. status is soft (unknown values map to
				// PAYMENT_STATUS_OTHER) — not asserted.
				Expect(tr.TargetCurrency).ToNot(BeEmpty())
				contracttest.AssertDecimalAmount(tr.TargetValue, "transfer targetValue")
				ids = append(ids, tr.ID)
			}

			// fillPayments skips transfer.ID <= LastTransferID and takes the
			// LAST item of the walk as the new watermark: the offset walk must
			// yield ascending IDs. If this ever goes red because Wise lists
			// newest-first, that is a latent connector watermark bug to
			// escalate, not a test to silence.
			contracttest.AssertNonDecreasing(ids, "transfer ids")
		})
	})

	Describe("quote + unfunded transfer + read-back", func() {
		// CreateQuote/CreateTransfer mirror createTransfer (transfers.go):
		// same-currency quote, then a transfer targeting a recipient account.
		// The transfer is never funded (the connector has no funding call), so
		// no money moves; Wise auto-expires unfunded transfers.
		It("creates a quote and an unfunded transfer, then reads it back by ID", func() {
			profile := contractProfile()

			page, err := c.GetRecipientAccounts(ctx, profile.ID, contractPageSize, 0)
			Expect(err).To(BeNil())
			if len(page.Content) == 0 {
				Skip("the sandbox profile has no recipient accounts — seed one via the sandbox UI to enable the recipient/transfer specs")
			}
			recipient := page.Content[0]

			quote, err := c.CreateQuote(ctx, fmt.Sprintf("%d", profile.ID), recipient.Currency, "1")
			Expect(err).To(BeNil())
			// createTransfer dereferences quote.ID.String() — a missing or
			// non-UUID id is a hard failure.
			Expect(quote.ID).ToNot(Equal(uuid.Nil))

			// customerTransactionId must be a UUID, unique per run so we never
			// hit an idempotency conflict.
			created, err := c.CreateTransfer(ctx, quote, recipient.ID, contracttest.UUIDRef())
			Expect(err).To(BeNil())
			Expect(created).ToNot(BeNil())

			// The fields fromTransferToPayment hard-consumes on the create
			// response (a nil error already proves the `created` date parse).
			Expect(created.ID).ToNot(BeZero())
			Expect(created.CreatedAt.IsZero()).To(BeFalse(), "created transfer has no created date")
			Expect(created.TargetCurrency).To(Equal(recipient.Currency))
			contracttest.AssertDecimalAmount(created.TargetValue, "created transfer targetValue")

			// Read-back: the surface TranslateTransferStateChangedWebhook
			// depends on (webhooks.go fetches the transfer by ID).
			got, err := c.GetTransfer(ctx, fmt.Sprintf("%d", created.ID))
			Expect(err).To(BeNil())
			Expect(got).ToNot(BeNil())
			Expect(got.ID).To(Equal(created.ID))
			Expect(got.TargetCurrency).To(Equal(recipient.Currency))
			contracttest.AssertDecimalAmount(got.TargetValue, "transfer targetValue")
		})
	})

	Describe("webhook subscription lifecycle", func() {
		// Mirrors createWebhooks (plugin.go): the connector installs BOTH
		// subscriptions, whose trigger/version pairs could drift
		// independently. Create → assert shape → list → delete.
		It("creates, lists and deletes the subscriptions the connector installs", func() {
			// Wise probes the callback URL with a POST at create time and
			// 422s (INVALID_CALLBACK_URL) unless it answers 2xx. Verify the
			// echo receiver is up first: its downtime must Skip this spec,
			// not masquerade as upstream drift.
			probe, perr := http.Post(contractWebhookURL, "application/json", strings.NewReader(`{}`))
			if perr != nil || probe.StatusCode >= 300 {
				Skip(fmt.Sprintf("webhook receiver %s is not answering POSTs with 2xx — Wise validates the callback URL at create time, so the lifecycle cannot be exercised", contractWebhookURL))
			}
			probe.Body.Close()

			profile := contractProfile()

			for _, sub := range []struct {
				name      string
				triggerOn string
				version   string
			}{
				{name: "transfer_state_changed", triggerOn: "transfers#state-change", version: "2.0.0"},
				{name: "balance_update", triggerOn: "balances#update", version: "2.2.0"},
			} {
				created, err := c.CreateWebhook(ctx, profile.ID, sub.name, sub.triggerOn, contractWebhookURL, sub.version)
				Expect(err).To(BeNil(), "failed to create %s subscription", sub.name)
				Expect(created).ToNot(BeNil())
				// createWebhooks consumes the id (PSPOther.ID) and uninstall
				// deletes by it.
				Expect(created.ID).ToNot(BeEmpty())

				subscriptionID := created.ID
				DeferCleanup(func() {
					derr := c.DeleteWebhooks(ctx, profile.ID, subscriptionID)
					Expect(derr).To(BeNil())
				})

				Expect(created.TriggerOn).To(Equal(sub.triggerOn))
				// uninstall matches the connector ID as a substring of
				// delivery.url — the URL must round-trip.
				Expect(created.Delivery.URL).To(Equal(contractWebhookURL))
			}

			// uninstall lists every profile's subscriptions and needs id +
			// delivery.url on each entry.
			listed, err := c.ListWebhooksSubscription(ctx, profile.ID)
			Expect(err).To(BeNil())
			Expect(listed).ToNot(BeEmpty())

			found := 0
			for _, s := range listed {
				Expect(s.ID).ToNot(BeEmpty())
				Expect(s.Delivery.URL).ToNot(BeEmpty())
				if s.Delivery.URL == contractWebhookURL {
					found++
				}
			}
			Expect(found).To(Equal(2), "both created subscriptions should be listed")
		})
	})
})
