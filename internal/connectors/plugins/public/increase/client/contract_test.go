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
	"os"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/contracttest"
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

// expectedAccountIDs / expectedTransactionIDs pin the known, seeded order of the
// sandbox's accounts and posted transactions. They start empty; the schema specs
// print the live IDs to stderr as a paste-ready Go literal (bootstrap log) so a
// maintainer can fill these in to enable the ordering contract.
//
// Only lists whose ORDER the connector consumes get a pin:
//   - accounts — fetchNextAccounts derives its watermark from list position
//     (LastCreatedAt = last account's created_at), so order is a dependency. The
//     suite never creates internal accounts, so a single-page subsequence is
//     stable and never needs re-bootstrapping.
//   - posted transactions — the client's Timeline scans reverse-chron, so order
//     is a dependency. The money-movement specs append new transactions at the
//     tail, so a growth-tolerant SUBSEQUENCE (contracttest.FilterToPinned) over
//     the OLDEST, already-settled IDs is used; the walk stops once the pins are
//     found, so it never pages into the new tail. Do NOT pin recent/unsettled IDs.
//     Only a small handful of the oldest IDs is pinned — enough to detect a
//     reordering without walking the whole (growing) history. The bootstrap log
//     prints the full list; copy just the first (oldest) few.
//
// External accounts are intentionally NOT pinned: fetchNextExternalAccounts
// paginates purely by opaque cursor with no watermark or positional assumption,
// so their order is not a connector contract — only their schema is asserted.
// This also avoids a growing pin (the suite creates one external account per run).
var (
	expectedAccountIDs = []string{
		"sandbox_account_ivm86k4k2mgjcdt1vl9e",
		"sandbox_account_ofldriuquj7pybufeiey",
		"sandbox_account_llq50lkrzjeaaccwslwu",
		"sandbox_account_j1u3pqtkfgsnx8maoayv",
		"sandbox_account_jej5yeeaclda0f8txisq",
		"sandbox_account_hjy3rdiuxqraisz9xv9e",
		"sandbox_account_oxniuygbaz3cbz61wfci",
		"sandbox_account_htjjym0uf6zmuf9dnaot",
		"sandbox_account_twiam0dikqw1ifu0xal4",
		"sandbox_account_zh8lecyazbh6vcf3qymo",
		"sandbox_account_9af0a3d1ldhn1gjcpcb2",
		"sandbox_account_ughhsp43drgjgytqfnuy",
		"sandbox_account_x03pxarcusyfcauimf1t",
	}
	expectedTransactionIDs = []string{
		"sandbox_transaction_pwv6ixzydpqqw5uueyqw",
		"sandbox_transaction_2bxj1cqxbxdfnb6aeg3s",
		"sandbox_transaction_mlpy0dftp1nt6t1vwq3z",
		"sandbox_transaction_heqms57g2vkelftb06hk",
		"sandbox_transaction_vpfw4ebxbqk7vb2nwlds",
		"sandbox_transaction_ltvoukzcmyzcafzymkc2",
		"sandbox_transaction_k5ga9kms7v0qccy7vybd",
		"sandbox_transaction_hdfi55zyulv6ze73t5cu",
		"sandbox_transaction_aaqlqfucp9zmk0tdutq6",
		"sandbox_transaction_w711z7j4tiyh4iwwxrlu",
	}
)

// accountIDs projects a fetched page of accounts to its IDs in list order.
// Accounts fit on a single page (page size 100) and the suite never creates
// internal accounts, so the ordering contract fetches one page rather than
// walking the whole list.
func accountIDs(accounts []*Account) []string {
	ids := make([]string, 0, len(accounts))
	for _, a := range accounts {
		ids = append(ids, a.ID)
	}
	return ids
}

// txListFn is one of GetTransactions / GetPendingTransactions /
// GetDeclinedTransactions — all share this signature.
type txListFn func(context.Context, int, Timeline) ([]*Transaction, Timeline, bool, error)

// guardPages bounds every Timeline walk against an accidental unbounded loop. It
// is a safety cap, not a fetch count: each walk stops as soon as the API reports
// no more pages (or, for the ordering check, once every pinned ID is found).
const guardPages = 2000

// walkTransactionIDs drives a fresh Timeline the way the connector does and
// returns transaction IDs in the chronological order it consumes them
// (oldest→newest). Increase's Timeline first scans back to the oldest record
// (emitting empty batches while it advances), then walks forward, so a single
// call is not enough. When until is non-nil the walk STOPS as soon as every ID
// in until has been collected: pinned IDs are the oldest, which the connector
// emits first, so the ordering check never pages into the growing new tail. With
// until nil it walks the full list (used only by the on-demand bootstrap log).
func walkTransactionIDs(ctx context.Context, fn txListFn, until []string) ([]string, error) {
	want := make(map[string]struct{}, len(until))
	for _, id := range until {
		want[id] = struct{}{}
	}

	var ids []string
	found := 0
	timeline := Timeline{}
	for i := 0; i < guardPages; i++ {
		batch, tl, hasMore, err := fn(ctx, contractPageSize, timeline)
		if err != nil {
			return nil, err
		}
		timeline = tl
		for _, tx := range batch {
			ids = append(ids, tx.ID)
			if _, ok := want[tx.ID]; ok {
				found++
			}
		}
		if len(until) > 0 && found >= len(want) {
			break
		}
		if !hasMore {
			break
		}
	}
	return ids, nil
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

		ctx = context.Background()
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

			if contracttest.BootstrapEnabled("INCREASE") {
				contracttest.LogBootstrap("expectedAccountIDs", accountIDs(accounts))
			}
		})

		It("keeps the known accounts in their expected, stable relative order", func() {
			if len(expectedAccountIDs) == 0 {
				Skip("expectedAccountIDs is not populated — set INCREASE_CONTRACT_BOOTSTRAP=1 and fill it from the bootstrap log to enable the ordering contract")
			}

			// Accounts fit on a single page (page size 100), so one fetch is
			// enough. Subsequence-match the pinned IDs against that page (ignoring
			// any out-of-band additions); re-bootstrap if the list ever exceeds a
			// page and the oldest pins spill off it.
			accounts, _, err := c.GetAccounts(ctx, contractPageSize, "", time.Time{})
			Expect(err).To(BeNil())
			gotKnownIDs := contracttest.FilterToPinned(accountIDs(accounts), expectedAccountIDs)
			Expect(gotKnownIDs).To(Equal(expectedAccountIDs))
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
			_, perr := balance.AvailableBalance.Int64()
			Expect(perr).To(BeNil(), "available_balance %q is not an integer", balance.AvailableBalance.String())
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

			if contracttest.BootstrapEnabled("INCREASE") {
				allIDs, err := walkTransactionIDs(ctx, c.GetTransactions, nil)
				Expect(err).To(BeNil())
				contracttest.LogBootstrap("expectedTransactionIDs", allIDs)
			}
		})

		It("keeps the known posted transactions in their expected, stable relative order", func() {
			if len(expectedTransactionIDs) == 0 {
				Skip("expectedTransactionIDs is not populated — set INCREASE_CONTRACT_BOOTSTRAP=1 and fill it from the bootstrap log to enable the ordering contract")
			}

			// The money-movement specs create new transactions each run, which the
			// oldest→newest walk returns at the tail. Assert only that the pinned
			// (old, settled) transactions retain their relative order, ignoring any
			// newly created ones. The walk stops once all pinned IDs are found, so it
			// never pages into the growing new tail.
			allIDs, err := walkTransactionIDs(ctx, c.GetTransactions, expectedTransactionIDs)
			Expect(err).To(BeNil())
			gotKnownIDs := contracttest.FilterToPinned(allIDs, expectedTransactionIDs)
			Expect(gotKnownIDs).To(Equal(expectedTransactionIDs))
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
