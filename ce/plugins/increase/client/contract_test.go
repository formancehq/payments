//go:build contract

// Package client contract test for the Increase connector.
//
// This is a CONTRACT test: it calls the real Increase sandbox over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types) or in list ordering. It is gated behind the `contract`
// build tag so it never runs as part of `just tests` (which only enables
// `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	INCREASE_CONTRACT_API_KEY=... just contract-tests increase
//
// The connector targets Increase's stable (unversioned) REST API. Sandbox and
// production are DIFFERENT hosts (sandbox.increase.com vs api.increase.com) with
// environment-specific keys, so the endpoint is hardcoded to the sandbox host
// (contractEndpoint) and only the API key is a secret. The webhookSharedSecret
// is only used as the fallback shared_secret when creating an event subscription
// (Increase accepts any string), so it is a non-secret test constant. Without
// INCREASE_CONTRACT_API_KEY the suite Skips rather than fails, so it is safe to
// run anywhere.
//
// Money movement: the InitiateTransfer / InitiateACHTransferPayout /
// CreateBankAccount specs make REAL calls against the sandbox at the smallest
// possible amount (1 minor unit) with a unique idempotency key per run. Increase
// has NO overdraft flag, so the money-movement specs source the account from a
// funded GetAccountBalance and Skip when no account is funded (or there are too
// few accounts / no external account). They accumulate sandbox state by design
// (accepted): Increase has no delete for external accounts, and each run creates
// one. Wire / RTP / Check payouts are intentionally NOT exercised: RTP and Check
// need a source_account_number_id (an Account Number resource) the client cannot
// list, so inputs can't be derived at runtime; their schema is covered by unit
// tests.
package client

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/contracttest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIncreaseContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Increase Contract Suite")
}

// contractPageSize is the connector's PAGE_SIZE (Increase's max is 100).
const contractPageSize = 100

// contractEndpoint is the Increase sandbox host. Sandbox and production are
// different hosts with different keys, so it is hardcoded to never risk pointing
// a sandbox key at production.
const contractEndpoint = "https://sandbox.increase.com"

// contractSharedSecret is passed to New as the client's webhookSharedSecret. It
// is only used as the fallback shared_secret when creating an event subscription
// (Increase accepts any string), so it is a non-secret test constant, not a CI
// secret.
const contractSharedSecret = "formance-contract-test-shared-secret"

// contractWebhookURL is a syntactically valid public HTTPS URL used only to
// create the temporary event subscription. NOTE: real end-to-end webhook
// *delivery* (Increase POSTing to this URL and us verifying its signature) is
// NOT exercised here — CI has no public ingress. This contract test only
// validates the management-API representation of the event subscription (create,
// shape, disable). Delivery/signature verification is covered by the unit tests.
const contractWebhookURL = "https://example.com/api/payments/v3/connectors/webhooks/increase-contract"

// contractMinAmount is the smallest amount the money-movement specs send: 1
// minor unit (1 cent). Increase amounts are integer minor units.
const contractMinAmount = int64(1)

// eventSubscriptionStatusDeleted mirrors the connector's disable path: Increase
// has no delete for event subscriptions, so uninstall sets status="deleted".
const eventSubscriptionStatusDeleted = "deleted"

// Ordering is asserted on the SORT KEY, not on pinned IDs. Only lists whose
// ORDER the connector consumes get an ordering spec:
//   - accounts — Increase returns accounts newest-first (reverse
//     chronological), and fetchNextAccounts derives its watermark from the
//     first (newest) account's created_at — asserted directly as a descending
//     sort key, with no pinned IDs to bootstrap or refill.
//   - posted transactions — the client's Timeline consumes chronological
//     (oldest→newest) order; same created_at non-decreasing assertion over
//     the full walk. created_at is immutable, so newly created transactions
//     (this suite's own money movement) appended at the tail can never
//     break the assertion.
//
// External accounts intentionally get NO ordering spec:
// fetchNextExternalAccounts paginates purely by opaque cursor with no
// watermark or positional assumption, so their order is not a connector
// contract — only their schema is asserted.

// txListFn is one of GetTransactions / GetPendingTransactions /
// GetDeclinedTransactions — all share this signature.
type txListFn func(context.Context, int, Timeline) ([]*Transaction, Timeline, bool, error)

// guardPages bounds every Timeline walk against an accidental unbounded loop. It
// is a safety cap, not a fetch count: each walk stops as soon as the API reports
// no more pages (or, for the ordering check, once every pinned ID is found).
const guardPages = 2000

// walkTransactionCreatedAts drives a fresh Timeline the way the connector does
// and returns each transaction's created_at sort key in the chronological
// order it consumes them (oldest→newest). Increase's Timeline first scans back
// to the oldest record (emitting empty batches while it advances), then walks
// forward, so a single call is not enough.
func walkTransactionCreatedAts(ctx context.Context, fn txListFn) ([]int64, error) {
	var keys []int64
	timeline := Timeline{}
	for i := 0; i < guardPages; i++ {
		batch, tl, hasMore, err := fn(ctx, contractPageSize, timeline)
		if err != nil {
			return nil, err
		}
		timeline = tl
		for _, tx := range batch {
			t, err := time.Parse(time.RFC3339, tx.CreatedAt)
			if err != nil {
				return nil, fmt.Errorf("transaction %s created_at %q is not RFC3339: %w", tx.ID, tx.CreatedAt, err)
			}
			keys = append(keys, t.UnixNano())
		}
		if !hasMore {
			break
		}
	}
	return keys, nil
}

// firstTransactionID returns the first transaction ID the connector would
// consume (stopping at the first non-empty batch), for the get-by-ID specs.
// ok=false when the list is empty.
func firstTransactionID(ctx context.Context, fn txListFn) (string, bool, error) {
	timeline := Timeline{}
	for i := 0; i < guardPages; i++ {
		batch, tl, hasMore, err := fn(ctx, contractPageSize, timeline)
		if err != nil {
			return "", false, err
		}
		timeline = tl
		if len(batch) > 0 {
			return batch[0].ID, true, nil
		}
		if !hasMore {
			break
		}
	}
	return "", false, nil
}

// findFundedAccount returns the ID of the first account whose available balance
// parses and is at least minAmount, so a minAmount movement won't hit
// insufficient funds (Increase has no overdraft). ok=false when none is funded.
func findFundedAccount(ctx context.Context, c Client, accounts []*Account, minAmount int64) (string, bool, error) {
	for _, a := range accounts {
		balance, _, err := c.GetAccountBalance(ctx, a.ID)
		if err != nil {
			return "", false, err
		}
		amount, perr := balance.AvailableBalance.Int64()
		if perr != nil {
			continue
		}
		if amount >= minAmount {
			return a.ID, true, nil
		}
	}
	return "", false, nil
}

var _ = Describe("Increase API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		apiKey := os.Getenv("INCREASE_CONTRACT_API_KEY")
		if apiKey == "" {
			Skip("INCREASE_CONTRACT_API_KEY must be set to run the Increase contract test")
		}

		// Bound each spec so a hung or slow sandbox call fails fast instead of
		// stalling the daily CI job indefinitely.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancel)
		c = New("increase", apiKey, contractEndpoint, contractSharedSecret)
	})

	Describe("GetAccounts", func() {
		It("returns accounts whose shape matches what the connector consumes", func() {
			accounts, _, err := c.GetAccounts(ctx, contractPageSize, "", time.Time{})
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			for _, a := range accounts {
				// id -> Reference (hard). created_at is hard: fillAccounts parses
				// it as RFC3339 and errors otherwise. currency is hard: it drives
				// DefaultAsset (FormatAsset), which balances reuse. name/type/bank/
				// status are tolerated (address-only / metadata), so NOT asserted.
				Expect(a.ID).ToNot(BeEmpty())
				_, perr := time.Parse(time.RFC3339, a.CreatedAt)
				Expect(perr).To(BeNil(), "account created_at %q is not RFC3339", a.CreatedAt)
				Expect(a.Currency).ToNot(BeEmpty())
			}

		})

		It("returns accounts in the created_at order the connector's watermark assumes", func() {
			// Increase returns accounts newest-first (reverse chronological), so
			// fetchNextAccounts watermarks on the FIRST account's created_at (the
			// max) — assert the sort key is descending directly. Accounts fit on a
			// single page (page size 100), so one fetch covers the list.
			accounts, _, err := c.GetAccounts(ctx, contractPageSize, "", time.Time{})
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			keys := make([]int64, 0, len(accounts))
			for _, a := range accounts {
				t, perr := time.Parse(time.RFC3339, a.CreatedAt)
				Expect(perr).To(BeNil(), "account created_at %q is not RFC3339", a.CreatedAt)
				keys = append(keys, t.UnixNano())
			}
			// Increase lists newest-first, so created_at must be non-increasing:
			// the connector's watermark relies on the first element being the max.
			for i := 1; i < len(keys); i++ {
				Expect(keys[i] <= keys[i-1]).To(BeTrue(),
					"accounts created_at is not sorted descending at index %d: %d > %d", i, keys[i], keys[i-1])
			}
		})
	})

	Describe("GetAccountBalance", func() {
		It("returns the balance of an account by ID with a numeric available balance", func() {
			accounts, _, err := c.GetAccounts(ctx, contractPageSize, "", time.Time{})
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			balance, _, err := c.GetAccountBalance(ctx, accounts[0].ID)
			Expect(err).To(BeNil())
			Expect(balance).ToNot(BeNil())

			// account_id -> AccountReference (hard). available_balance is hard: it
			// is parsed via big.Int.SetString(s, 10), so it must be an integer
			// minor-unit string. current_balance / type are unused by the
			// connector, so they are NOT asserted.
			Expect(balance.AccountID).ToNot(BeEmpty())
			contracttest.AssertIntegerAmount(balance.AvailableBalance, "available_balance")
		})
	})

	Describe("GetExternalAccounts", func() {
		It("returns external accounts whose shape matches what the connector consumes", func() {
			accounts, _, err := c.GetExternalAccounts(ctx, contractPageSize, "")
			Expect(err).To(BeNil())

			for _, a := range accounts {
				// id -> Reference (hard). created_at is hard: mapExternalAccount
				// parses it as RFC3339 and errors otherwise. description/type/
				// account_holder/account_number/status/routing_number are tolerated
				// (address-only / metadata), so they are NOT asserted.
				Expect(a.ID).ToNot(BeEmpty())
				_, perr := time.Parse(time.RFC3339, a.CreatedAt)
				Expect(perr).To(BeNil(), "external account created_at %q is not RFC3339", a.CreatedAt)
			}
			// No ordering spec: fetchNextExternalAccounts paginates purely by opaque
			// cursor with no watermark/positional assumption, so external-account
			// order is not a connector contract (only the schema above is).
		})
	})

	Describe("GetTransactions", func() {
		It("returns posted transactions whose shape matches what the connector consumes", func() {
			// A fresh Timeline drives the client's backlog scan; the first call may
			// return empty while still scanning, so we only validate whatever comes
			// back (schema, not presence).
			transactions, _, _, err := c.GetTransactions(ctx, contractPageSize, Timeline{})
			Expect(err).To(BeNil())

			assertTransactionShape(transactions)

		})

		It("returns posted transactions in the chronological order the connector consumes", func() {
			// The Timeline walk consumes oldest→newest order; assert the
			// created_at sort key is honored across the full walk. created_at
			// is immutable, so transactions this suite creates (appended at
			// the tail) can never break the assertion.
			keys, err := walkTransactionCreatedAts(ctx, c.GetTransactions)
			Expect(err).To(BeNil())
			Expect(keys).ToNot(BeEmpty())
			contracttest.AssertNonDecreasing(keys, "posted transactions created_at")
		})

		It("gets a posted transaction by ID with a valid shape", func() {
			id, ok, err := firstTransactionID(ctx, c.GetTransactions)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no posted transactions in the sandbox to exercise GetTransaction")
			}

			tx, err := c.GetTransaction(ctx, id)
			Expect(err).To(BeNil())
			assertTransactionShape([]*Transaction{tx})
		})
	})

	Describe("GetPendingTransactions", func() {
		It("returns pending transactions whose shape matches what the connector consumes", func() {
			transactions, _, _, err := c.GetPendingTransactions(ctx, contractPageSize, Timeline{})
			Expect(err).To(BeNil())
			assertTransactionShape(transactions)
		})

		It("gets a pending transaction by ID with a valid shape", func() {
			id, ok, err := firstTransactionID(ctx, c.GetPendingTransactions)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no pending transactions in the sandbox to exercise GetPendingTransaction")
			}

			tx, err := c.GetPendingTransaction(ctx, id)
			Expect(err).To(BeNil())
			assertTransactionShape([]*Transaction{tx})
		})
	})

	Describe("GetDeclinedTransactions", func() {
		It("returns declined transactions whose shape matches what the connector consumes", func() {
			transactions, _, _, err := c.GetDeclinedTransactions(ctx, contractPageSize, Timeline{})
			Expect(err).To(BeNil())
			assertTransactionShape(transactions)
		})

		It("gets a declined transaction by ID with a valid shape", func() {
			id, ok, err := firstTransactionID(ctx, c.GetDeclinedTransactions)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no declined transactions in the sandbox to exercise GetDeclinedTransaction")
			}

			tx, err := c.GetDeclinedTransaction(ctx, id)
			Expect(err).To(BeNil())
			assertTransactionShape([]*Transaction{tx})
		})
	})

	Describe("ListEventSubscriptions", func() {
		It("returns event subscriptions whose shape matches what the connector consumes", func() {
			subscriptions, err := c.ListEventSubscriptions(ctx)
			Expect(err).To(BeNil())

			for _, s := range subscriptions {
				// id and url are consumed by uninstall (matched against the
				// connector ID and disabled by ID).
				Expect(s.ID).ToNot(BeEmpty())
				Expect(s.URL).ToNot(BeEmpty())
			}
		})
	})

	Describe("event subscription lifecycle", func() {
		// Increase has no delete for event subscriptions; uninstall disables them
		// by setting status="deleted" (UpdateEventSubscription). We create one and
		// disable it in DeferCleanup — a clean, self-reclaiming pair.
		It("creates an event subscription with a valid, retrievable shape, then disables it", func() {
			created, err := c.CreateEventSubscription(ctx, &CreateEventSubscriptionRequest{
				SelectedEventCategory: string(EventCategoryTransactionCreated),
				SharedSecret:          contractSharedSecret,
				URL:                   contractWebhookURL,
			}, contracttest.Ref("increase", "webhook"))
			Expect(err).To(BeNil())
			Expect(created).ToNot(BeNil())
			Expect(created.ID).ToNot(BeEmpty())

			DeferCleanup(func() {
				_, derr := c.UpdateEventSubscription(ctx, &UpdateEventSubscriptionRequest{
					Status: eventSubscriptionStatusDeleted,
				}, created.ID)
				Expect(derr).To(BeNil())
			})

			Expect(created.URL).To(Equal(contractWebhookURL))
			Expect(created.Status).ToNot(BeEmpty())
		})
	})

	Describe("InitiateTransfer", func() {
		// Internal account transfer between two of our own accounts: money stays
		// on the platform. Increase has no overdraft, so we source from a funded
		// account and Skip when none is funded. Smallest amount, unique
		// idempotency key per run.
		It("initiates a minimal internal transfer", func() {
			accounts, _, err := c.GetAccounts(ctx, contractPageSize, "", time.Time{})
			Expect(err).To(BeNil())
			if len(accounts) < 2 {
				Skip("need at least 2 accounts in the sandbox to exercise a transfer")
			}

			source, ok, err := findFundedAccount(ctx, c, accounts, contractMinAmount)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no funded account to source a transfer (Increase has no overdraft)")
			}

			var destination string
			for _, a := range accounts {
				if a.ID != source {
					destination = a.ID
					break
				}
			}
			if destination == "" {
				Skip("no distinct destination account to exercise a transfer")
			}

			resp, err := c.InitiateTransfer(ctx, &TransferRequest{
				AccountID:            source,
				DestinationAccountID: destination,
				Amount:               contractMinAmount,
				Description:          "Formance Contract Test",
			}, contracttest.Ref("increase", "transfer"))
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
			Expect(resp.Amount).To(Equal(contractMinAmount))
			_, perr := time.Parse(time.RFC3339, resp.CreatedAt)
			Expect(perr).To(BeNil(), "transfer created_at %q is not RFC3339", resp.CreatedAt)
		})
	})

	Describe("CreateBankAccount", func() {
		// Increase has no delete for external accounts, so each run accumulates
		// one external account in the sandbox. This is accepted (small, sandbox
		// only) and also guarantees an ACH payout destination exists.
		It("creates an external account whose shape matches what the connector consumes", func() {
			resp, err := c.CreateBankAccount(ctx, &BankAccountRequest{
				AccountNumber: "1234567890",
				RoutingNumber: "121000248", // checksum-valid ABA (Wells Fargo)
				AccountHolder: "business",
				Description:   "Formance Contract Test",
			}, contracttest.Ref("increase", "external"))
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.RoutingNumber).ToNot(BeEmpty())
			_, perr := time.Parse(time.RFC3339, resp.CreatedAt)
			Expect(perr).To(BeNil(), "external account created_at %q is not RFC3339", resp.CreatedAt)
		})
	})

	Describe("InitiateACHTransferPayout", func() {
		// Outbound ACH payout to an external account. Funded source derived from a
		// GetAccountBalance; destination from GetExternalAccounts. Smallest amount,
		// unique idempotency key per run.
		It("initiates a minimal ACH payout", func() {
			accounts, _, err := c.GetAccounts(ctx, contractPageSize, "", time.Time{})
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			source, ok, err := findFundedAccount(ctx, c, accounts, contractMinAmount)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no funded account to source an ACH payout (Increase has no overdraft)")
			}

			externalAccounts, _, err := c.GetExternalAccounts(ctx, contractPageSize, "")
			Expect(err).To(BeNil())
			if len(externalAccounts) == 0 {
				Skip("need at least 1 external account in the sandbox to exercise an ACH payout")
			}

			resp, err := c.InitiateACHTransferPayout(ctx, &ACHPayoutRequest{
				AccountID:         source,
				ExternalAccountID: externalAccounts[0].ID,
				Amount:            contractMinAmount,
				// individual_name must be non-empty: Increase rejects an empty
				// string (Minimum length is 1). In production the connector sets it
				// from the destination account name.
				IndividualName:      "Formance Contract Test",
				StatementDescriptor: "Formance CT",
			}, contracttest.Ref("increase", "ach"))
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
		})
	})
})

// assertTransactionShape validates the fields the connector hard-depends on for
// every transaction in the batch: id -> Reference; created_at parses as RFC3339
// (mapPayment errors otherwise); currency drives FormatAsset. amount is int64 —
// a non-numeric value would fail JSON unmarshal upstream, so a decoded batch
// already proves that contract. source.category / account_id / route_* are
// tolerated (metadata / address-only), so they are NOT asserted.
func assertTransactionShape(transactions []*Transaction) {
	for _, tx := range transactions {
		Expect(tx.ID).ToNot(BeEmpty())
		_, perr := time.Parse(time.RFC3339, tx.CreatedAt)
		Expect(perr).To(BeNil(), "transaction created_at %q is not RFC3339", tx.CreatedAt)
		Expect(tx.Currency).ToNot(BeEmpty())
	}
}
