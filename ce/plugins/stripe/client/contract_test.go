//go:build contract

// Package client contract test for the Stripe connector.
//
// This is a CONTRACT test: it calls the real Stripe API in TEST MODE over the
// network through the same client.Client the connector uses, and asserts that
// the responses the Payments project depends on have not drifted in schema
// (field presence + types) or in list ordering. It is gated behind the
// `contract` build tag so it never runs as part of `just tests` (which only
// enables `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	STRIPE_CONTRACT_API_KEY=sk_test_... just contract-tests stripe
//
// Auth & endpoint: Stripe's sandbox is test mode on the SAME host as prod
// (api.stripe.com), selected purely by the key prefix (sk_test_ vs sk_live_),
// so the client's built-in default backend is used and only the API key is a
// secret. The key MUST be an sk_test_ key. Without it the suite Skips rather
// than fails. client.New itself calls GET /v1/account, so constructing the
// client is already a contract touch. The SDK (stripe-go/v80) pins API version
// 2024-09-30.acacia via the Stripe-Version header; Stripe does not sunset
// dated API versions, so this surface is stable (see the spec's API-currency
// note).
//
// Ordering: the connector's Timeline is positional for accounts, external
// accounts and balance transactions alike — the backlog walk pages with
// StartingAfter assuming the API lists newest-first, and the caught-up walk
// reverses each page and takes the last element as the new LatestID. The real
// dependency is therefore the SORT PROPERTY ("Stripe lists newest-first"),
// asserted monotonically on Created where a timestamp exists (accounts,
// balance transactions — nothing to bootstrap). stripe.BankAccount has no
// timestamp, so external-account ordering is the one PINNED-ID subsequence:
// expectedExternalAccountIDs starts empty (the spec Skips) and is filled from
// the STRIPE_CONTRACT_BOOTSTRAP=1 log once sandbox credentials exist.
//
// Scope: all client.Client methods are consumed by the plugin, so all are
// covered — reads (root account, connected accounts, balances, bank accounts,
// balance transactions), money movement (a self-cleaning 1-minor-unit
// transfer→full-reversal pair, plus a payout gated on a funded connected
// account owning a bank account), and the webhook-endpoint create→delete
// lifecycle (per-run-unique URL so the create path — the only one that returns
// the Secret — always runs).
package client

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"

	logging "github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/pkg/domain/contracttest"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v80"
)

func TestStripeContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stripe Contract Suite")
}

// contractPageSize mirrors the connector's PAGE_SIZE (Stripe's documented max
// page size is 100; the parent stripe package cannot be imported from here
// without an import cycle). The collectAll* walkers page through the full list
// regardless, so the value only affects round-trips.
const contractPageSize = 100

// contractDescription labels the records the money-movement specs create.
const contractDescription = "Formance contract test"

// contractPayoutAmount is the smallest amount Stripe accepts for a PAYOUT —
// verified live: "Amount must be no less than €1.00" (the contract sandbox is
// an FR platform, so everything is EUR). Transfers have no such floor: the
// transfer spec still moves 1 minor unit.
const contractPayoutAmount int64 = 100

// contractWebhookBase is the base of the per-run-unique webhook URL. It only
// needs to be syntactically valid and unique: real delivery/HMAC verification
// is covered by unit tests with recorded payloads (CI has no public ingress);
// this suite validates the management-API representation only.
const contractWebhookBase = "https://contract-tests.formance.dev/stripe"

// expectedExternalAccountIDs pins the relative order of the seeded external
// (bank) accounts across the connected-account walk. BankAccounts carry no
// timestamp, so this subsequence pin is the only way to assert the positional
// EndingBefore/LatestID ordering contract. Starts empty (the ordering spec
// Skips); fill it from a STRIPE_CONTRACT_BOOTSTRAP=1 run once the sandbox is
// seeded.
var expectedExternalAccountIDs = []string{
	"ba_1ToinCIQqyAoarXVfU9o5on2",
	"ba_1ToinAIQqyAoarXVOFPdOWrk",
	"ba_1ToiaxIQqyAoarXVLZatRKt2",
}

// collectAllAccounts walks the connected-account list exactly the way
// fetchNextAccounts does on a fresh install: an empty Timeline triggers the
// backlog walk (StartingAfter pages, NEWEST-FIRST across the whole walk) until
// the API reports no next page. Bounded to avoid an accidental unbounded loop.
func collectAllAccounts(ctx context.Context, c Client) ([]*stripe.Account, error) {
	var all []*stripe.Account
	var timeline Timeline
	for i := 0; i <= 5000; i++ {
		batch, tl, hasMore, err := c.GetAccounts(ctx, timeline, contractPageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		timeline = tl
		if !hasMore {
			return all, nil
		}
	}
	return all, nil
}

// collectAllExternalAccounts walks one connected account's bank accounts the
// way fetchNextExternalAccounts does (backlog walk, newest-first). Bounded to
// avoid an accidental unbounded loop.
func collectAllExternalAccounts(ctx context.Context, c Client, accountID string) ([]*stripe.BankAccount, error) {
	var all []*stripe.BankAccount
	var timeline Timeline
	for i := 0; i <= 5000; i++ {
		batch, tl, hasMore, err := c.GetExternalAccounts(ctx, accountID, timeline, contractPageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		timeline = tl
		if !hasMore {
			return all, nil
		}
	}
	return all, nil
}

// collectAllPayments walks an account's balance transactions the way
// fetchNextPayments does: the first call scans to the OLDEST transaction, then
// the walk advances chronologically via EndingBefore, so the collected slice
// is expected oldest-first. Bounded to avoid an accidental unbounded loop.
func collectAllPayments(ctx context.Context, c Client, accountID string) ([]*stripe.BalanceTransaction, error) {
	var all []*stripe.BalanceTransaction
	var timeline Timeline
	for i := 0; i <= 5000; i++ {
		batch, tl, hasMore, err := c.GetPayments(ctx, accountID, timeline, contractPageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		timeline = tl
		if !hasMore {
			return all, nil
		}
	}
	return all, nil
}

// assertPaymentSource asserts the per-type expanded source objects that
// translatePayment DEREFERENCES (a nil there panics the ingestion workflow) or
// errors on. A nil top-level Source is tolerated per row (translatePayment
// logs and skips it), so the caller counts those instead.
//
// Note on charge: translatePayment nil-checks Source.Charge for type "charge"
// (skip+log), but NOT for type "payment" (deref → panic). We assert it for
// both anyway: a nil expanded Charge on a charge-typed row means the
// AddExpand("data.source.charge") contract broke wholesale and every charge
// would be dropped silently — exactly the drift this suite exists to catch.
func assertPaymentSource(bt *stripe.BalanceTransaction) {
	switch bt.Type {
	case stripe.BalanceTransactionTypeCharge, stripe.BalanceTransactionTypePayment:
		Expect(bt.Source.Charge).ToNot(BeNil(), "balance transaction %s (%s) has no expanded charge", bt.ID, bt.Type)
	case stripe.BalanceTransactionTypeRefund,
		stripe.BalanceTransactionTypeRefundFailure,
		stripe.BalanceTransactionTypePaymentRefund:
		Expect(bt.Source.Refund).ToNot(BeNil(), "balance transaction %s (%s) has no expanded refund", bt.ID, bt.Type)
		Expect(bt.Source.Refund.Charge).ToNot(BeNil(), "refund on %s has no charge", bt.ID)
		Expect(bt.Source.Refund.Charge.BalanceTransaction).ToNot(BeNil(), "refund charge on %s has no balance transaction (ParentReference source)", bt.ID)
	case stripe.BalanceTransactionTypePaymentFailureRefund:
		// A missing refund source is tolerated (stored without a parent), but
		// when present its charge chain is dereferenced.
		if bt.Source.Refund != nil {
			Expect(bt.Source.Refund.Charge).ToNot(BeNil(), "refund on %s has no charge", bt.ID)
			Expect(bt.Source.Refund.Charge.BalanceTransaction).ToNot(BeNil(), "refund charge on %s has no balance transaction", bt.ID)
		}
	case stripe.BalanceTransactionTypePayout:
		Expect(bt.Source.Payout).ToNot(BeNil(), "balance transaction %s (payout) has no expanded payout", bt.ID)
	case stripe.BalanceTransactionTypePayoutFailure, stripe.BalanceTransactionTypePayoutCancel:
		Expect(bt.Source.Payout).ToNot(BeNil(), "balance transaction %s (%s) has no expanded payout", bt.ID, bt.Type)
		Expect(bt.Source.Payout.BalanceTransaction).ToNot(BeNil(), "payout on %s has no balance transaction (ParentReference source)", bt.ID)
	case stripe.BalanceTransactionTypeTransfer:
		Expect(bt.Source.Transfer).ToNot(BeNil(), "balance transaction %s (transfer) has no expanded transfer", bt.ID)
	case stripe.BalanceTransactionTypeTransferRefund,
		stripe.BalanceTransactionTypeTransferCancel,
		stripe.BalanceTransactionTypeTransferFailure:
		Expect(bt.Source.Transfer).ToNot(BeNil(), "balance transaction %s (%s) has no expanded transfer", bt.ID, bt.Type)
		Expect(bt.Source.Transfer.BalanceTransaction).ToNot(BeNil(), "transfer on %s has no balance transaction (ParentReference source)", bt.ID)
	case stripe.BalanceTransactionTypeAdjustment:
		// A missing dispute source is tolerated (skip+log), but when present
		// its charge chain is dereferenced.
		if bt.Source.Dispute != nil {
			Expect(bt.Source.Dispute.Charge).ToNot(BeNil(), "dispute on %s has no charge", bt.ID)
			Expect(bt.Source.Dispute.Charge.BalanceTransaction).ToNot(BeNil(), "dispute charge on %s has no balance transaction", bt.ID)
		}
	default:
		// Unsupported types are logged and skipped by translatePayment; their
		// shape is not a contract.
	}
}

// firstFundedAvailable returns the first Available entry of the balance with
// at least 1 minor unit, so a 1-minor-unit movement can be sourced from it
// (Stripe transfers/payouts have no overdraft). ok=false when nothing is
// funded (the caller then Skips).
func firstFundedAvailable(balance *stripe.Balance) (*stripe.Amount, bool) {
	for _, av := range balance.Available {
		if av == nil {
			continue
		}
		if av.Amount >= 1 && av.Currency != "" {
			return av, true
		}
	}
	return nil, false
}

var _ = Describe("Stripe API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		apiKey := os.Getenv("STRIPE_CONTRACT_API_KEY")
		if apiKey == "" {
			Skip("STRIPE_CONTRACT_API_KEY must be set to run the Stripe contract test")
		}

		ctx = context.Background()
		// nil backend => the SDK default (api.stripe.com); test mode is
		// selected by the sk_test_ key, exactly like production selects live.
		// New itself calls GET /v1/account and errors on a drifted/unauthorized
		// root-account response.
		var err error
		c, err = New("stripe", logging.NewDefaultLogger(GinkgoWriter, true, false, false), nil, apiKey)
		Expect(err).To(BeNil())
	})

	Describe("GetRootAccount", func() {
		It("returns the platform account whose shape matches what the connector consumes", func() {
			root, err := c.GetRootAccount()
			Expect(err).To(BeNil())

			// ID is a HARD dependency (PSPAccount.Reference, and the root-vs-
			// connected discriminator everywhere). Created is NOT asserted:
			// verified live, GET /v1/account OMITS created on the own-account
			// retrieve (it decodes to 0), so the connector already ingests the
			// root account as 1970-01-01 — a tolerated quirk, not a contract
			// (connected accounts DO carry created; see GetAccounts).
			// Settings.Dashboard.DisplayName and DefaultCurrency are tolerated
			// (nil-checked / FormatAsset), so they are NOT asserted either.
			Expect(root.ID).ToNot(BeEmpty())
			Expect(root.ID).To(Equal(c.GetRootAccountID()))
		})
	})

	Describe("GetAccounts", func() {
		It("returns connected accounts newest-first with the shape the connector consumes", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())
			if len(accounts) == 0 {
				Skip("no connected accounts in the test-mode account to exercise GetAccounts")
			}

			createds := make([]int64, 0, len(accounts))
			for _, a := range accounts {
				// ID -> PSPAccount.Reference and the Timeline cursor (hard).
				// Created -> time.Unix (silently 1970 when dropped, so assert).
				Expect(a.ID).ToNot(BeEmpty())
				Expect(a.Created).To(BeNumerically(">", 0))
				createds = append(createds, a.Created)
			}

			// The backlog walk pages StartingAfter and takes the first element
			// of the first page as its starting point, and the caught-up walk
			// reverses pages before deriving LatestID — both assume the API
			// lists NEWEST-FIRST, so non-increasing Created is the ordering
			// contract.
			slices.Reverse(createds)
			contracttest.AssertNonDecreasing(createds, "connected account created (oldest-first after reversal)")
		})
	})

	Describe("GetAccountBalances", func() {
		It("returns the root balance with the shape the connector consumes", func() {
			// Ingestion passes the root account's REAL ID through
			// resolveAccount (only the legacy "root" reference maps to ""), so
			// the balance call carries Stripe-Account: <root id>; mirror that.
			balance, err := c.GetAccountBalances(ctx, c.GetRootAccountID())
			Expect(err).To(BeNil())

			// An empty Available list would make ingestion silently emit no
			// balances, so its presence is the contract. Amount is an
			// SDK-typed int64 (big.NewInt consumes any value); Currency feeds
			// FormatAsset and the PSPBalance asset, so it must be present.
			Expect(balance.Available).ToNot(BeEmpty())
			for _, av := range balance.Available {
				Expect(av).ToNot(BeNil())
				Expect(string(av.Currency)).ToNot(BeEmpty())
			}
		})
	})

	Describe("GetExternalAccounts", func() {
		It("returns bank accounts whose shape matches what the connector consumes", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())

			var all []*stripe.BankAccount
			for _, a := range accounts {
				got, gerr := collectAllExternalAccounts(ctx, c, a.ID)
				Expect(gerr).To(BeNil())
				all = append(all, got...)
			}
			if len(all) == 0 {
				Skip("no connected account with external bank accounts to exercise GetExternalAccounts")
			}

			ids := make([]string, 0, len(all))
			for _, ba := range all {
				// ID -> PSPAccount.Reference and the Timeline cursor (hard).
				// Account is a HARD dependency: fetchNextExternalAccounts
				// ERRORS (halting ingestion) when it is nil. Account.Created
				// and Currency are tolerated (fallback to now / FormatAsset).
				Expect(ba.ID).ToNot(BeEmpty())
				Expect(ba.Account).ToNot(BeNil(), "bank account %s is missing its owning account", ba.ID)
				Expect(ba.Account.ID).ToNot(BeEmpty())
				ids = append(ids, ba.ID)
			}

			if contracttest.BootstrapEnabled("STRIPE") {
				contracttest.LogBootstrap("expectedExternalAccountIDs", ids)
			}
		})

		It("keeps the pinned external accounts in relative order", func() {
			if len(expectedExternalAccountIDs) == 0 {
				Skip("expectedExternalAccountIDs is not populated — run once with " +
					"STRIPE_CONTRACT_BOOTSTRAP=1 and paste the printed literal to enable " +
					"the ordering contract (BankAccounts have no timestamp, so a pinned " +
					"subsequence is the only way to assert the positional Timeline order)")
			}

			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())

			gotIDs := make([]string, 0, len(expectedExternalAccountIDs))
			for _, a := range accounts {
				got, gerr := collectAllExternalAccounts(ctx, c, a.ID)
				Expect(gerr).To(BeNil())
				for _, ba := range got {
					gotIDs = append(gotIDs, ba.ID)
				}
			}

			// Growth-tolerant subsequence: newly added bank accounts are
			// ignored; the pinned ones must retain their relative order.
			Expect(contracttest.FilterToPinned(gotIDs, expectedExternalAccountIDs)).
				To(Equal(expectedExternalAccountIDs))
		})
	})

	Describe("GetPayments", func() {
		It("returns balance transactions chronologically with the shapes the connector consumes", func() {
			// Mirror ingestion: the root account's payments are fetched with
			// its real ID as Stripe-Account (see GetAccountBalances note).
			txns, err := collectAllPayments(ctx, c, c.GetRootAccountID())
			Expect(err).To(BeNil())
			if len(txns) == 0 {
				Skip("no balance transactions on the root account to exercise GetPayments")
			}

			createds := make([]int64, 0, len(txns))
			withSource := 0
			for _, bt := range txns {
				// ID -> PSPPayment.Reference and the Timeline cursor (hard).
				// Created -> time.Unix + the walk's chronology (hard). Type
				// drives the translatePayment switch (hard).
				Expect(bt.ID).ToNot(BeEmpty())
				Expect(bt.Created).To(BeNumerically(">", 0))
				Expect(string(bt.Type)).ToNot(BeEmpty())
				createds = append(createds, bt.Created)

				if bt.Source == nil {
					// Tolerated per row (translatePayment logs and skips), but
					// counted: if EVERY row lost its source, ingestion would
					// silently drop everything.
					continue
				}
				withSource++
				assertPaymentSource(bt)
			}
			Expect(withSource).To(BeNumerically(">", 0),
				"every balance transaction came back without a source — the data.source expansion contract broke")

			// GetPayments scans to the OLDEST transaction and then walks
			// forward via EndingBefore, reversing each page — the collected
			// slice must therefore be chronological (non-decreasing Created).
			contracttest.AssertNonDecreasing(createds, "balance transaction created")
		})
	})

	Describe("CreateTransfer + ReverseTransfer", func() {
		// A self-cleaning pair: move 1 minor unit root -> connected account,
		// then reverse it in full — net balance change zero, and the reversal
		// (which needs a fresh transfer and is otherwise hard to make
		// repeatable) always has a valid target. Stripe has no overdraft for
		// transfers, so the spec is gated on a funded root balance, and on a
		// connected account with the transfers capability active.
		It("creates a 1-minor-unit transfer and fully reverses it", func() {
			balance, err := c.GetAccountBalances(ctx, c.GetRootAccountID())
			Expect(err).To(BeNil())
			funded, ok := firstFundedAvailable(balance)
			if !ok {
				Skip("root account has no available balance to source a transfer (no overdraft on transfers)")
			}

			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())
			var destination string
			for _, a := range accounts {
				if a.Capabilities != nil && a.Capabilities.Transfers == stripe.AccountCapabilityStatusActive {
					destination = a.ID
					break
				}
			}
			if destination == "" {
				Skip("no connected account with an active transfers capability to receive a transfer")
			}

			// Source nil = the root/platform account, as createTransfer sends
			// it. Unique idempotency key per run (Stripe allows <=255 chars):
			// always creates fresh, never a stale idempotency conflict.
			transfer, err := c.CreateTransfer(ctx, &CreateTransferRequest{
				IdempotencyKey: contracttest.Ref("stripe", "transfer"),
				Amount:         1,
				Currency:       string(funded.Currency),
				Source:         nil,
				Destination:    destination,
				Description:    contractDescription,
			})
			Expect(err).To(BeNil())
			Expect(transfer.ID).ToNot(BeEmpty())
			// BalanceTransaction is a HARD dependency: fromTransferToPayment
			// DEREFERENCES it for the payment Reference — a broken
			// balance_transaction expansion panics the workflow.
			Expect(transfer.BalanceTransaction).ToNot(BeNil())
			Expect(transfer.BalanceTransaction.ID).ToNot(BeEmpty())
			Expect(transfer.Created).To(BeNumerically(">", 0))
			Expect(transfer.Amount).To(Equal(int64(1)))
			Expect(string(transfer.Currency)).ToNot(BeEmpty())

			reversal, err := c.ReverseTransfer(ctx, ReverseTransferRequest{
				IdempotencyKey:   contracttest.Ref("stripe", "reversal"),
				StripeTransferID: transfer.ID,
				Account:          nil,
				Amount:           1,
				Description:      contractDescription,
			})
			Expect(err).To(BeNil())
			// fromTransferReversalToPayment dereferences BOTH expansions:
			// reversal.BalanceTransaction.ID (Reference) and
			// reversal.Transfer.BalanceTransaction.ID (ParentReference).
			Expect(reversal.BalanceTransaction).ToNot(BeNil())
			Expect(reversal.BalanceTransaction.ID).ToNot(BeEmpty())
			Expect(reversal.Transfer).ToNot(BeNil())
			Expect(reversal.Transfer.BalanceTransaction).ToNot(BeNil())
			Expect(reversal.Transfer.BalanceTransaction.ID).ToNot(BeEmpty())
			Expect(reversal.Created).To(BeNumerically(">", 0))
			Expect(reversal.Amount).To(Equal(int64(1)))
			Expect(string(reversal.Currency)).ToNot(BeEmpty())
		})
	})

	Describe("CreatePayout", func() {
		// A payout needs a destination BANK ACCOUNT on the source account, and
		// the root's bank accounts are unreachable through this client
		// (GetExternalAccounts returns empty for the root), so the source is a
		// connected account that owns a bank account. Payouts DRAIN the
		// connected account's balance run after run (unlike the self-cleaning
		// transfer+reversal pair), so the spec is SELF-FUNDING: when the
		// account is short of the €1.00 payout minimum it first tops itself up
		// from the platform balance via the connector's own CreateTransfer,
		// and only Skips when the platform is broke too. Test mode moves no
		// real money; records accumulate like Column/Increase, accepted.
		It("initiates a minimum payout from a connected account to its own bank account", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())

			var source, destination, payoutCurrency string
			for _, a := range accounts {
				bankAccounts, gerr := collectAllExternalAccounts(ctx, c, a.ID)
				if gerr != nil || len(bankAccounts) == 0 {
					continue
				}
				source = a.ID
				destination = bankAccounts[0].ID
				payoutCurrency = string(bankAccounts[0].Currency)
				break
			}
			if source == "" {
				Skip("no connected account owning a bank account to exercise a payout")
			}

			available := func(accountID string) int64 {
				balance, berr := c.GetAccountBalances(ctx, accountID)
				Expect(berr).To(BeNil())
				for _, av := range balance.Available {
					if av != nil && string(av.Currency) == payoutCurrency {
						return av.Amount
					}
				}
				return 0
			}

			if available(source) < contractPayoutAmount {
				if available(c.GetRootAccountID()) < contractPayoutAmount {
					Skip("neither the connected account nor the platform holds enough " +
						payoutCurrency + " to fund a minimum payout (no overdraft on payouts)")
				}
				_, terr := c.CreateTransfer(ctx, &CreateTransferRequest{
					IdempotencyKey: contracttest.Ref("stripe", "payout-funding"),
					Amount:         contractPayoutAmount,
					Currency:       payoutCurrency,
					Source:         nil,
					Destination:    source,
					Description:    contractDescription,
				})
				Expect(terr).To(BeNil())
			}

			payout, err := c.CreatePayout(ctx, &CreatePayoutRequest{
				IdempotencyKey: contracttest.Ref("stripe", "payout"),
				Amount:         contractPayoutAmount,
				Currency:       payoutCurrency,
				Source:         &source,
				Destination:    destination,
				Description:    contractDescription,
			})
			Expect(err).To(BeNil())
			Expect(payout.ID).ToNot(BeEmpty())
			// BalanceTransaction is a HARD dependency: fromPayoutToPayment
			// DEREFERENCES it for the payment Reference.
			Expect(payout.BalanceTransaction).ToNot(BeNil())
			Expect(payout.BalanceTransaction.ID).ToNot(BeEmpty())
			Expect(payout.Created).To(BeNumerically(">", 0))
			Expect(payout.Amount).To(Equal(contractPayoutAmount))
			Expect(string(payout.Currency)).ToNot(BeEmpty())
			// Status feeds matchPayoutStatus (hard).
			Expect(string(payout.Status)).ToNot(BeEmpty())
		})
	})

	Describe("webhook endpoint lifecycle", func() {
		It("creates the root and connect balance endpoints and deletes them", func() {
			// A per-run-unique base URL forces the CREATE path: for an
			// existing URL CreateWebhookEndpoints only UPDATES the endpoint,
			// and Stripe returns the signing Secret exclusively on create —
			// the field the whole verification path hangs on.
			base := contractWebhookBase + "/" + contracttest.UUIDRef()

			endpoints, err := c.CreateWebhookEndpoints(ctx, base)
			DeferCleanup(func() {
				configs := make([]models.PSPWebhookConfig, 0, len(endpoints))
				for _, e := range endpoints {
					configs = append(configs, models.PSPWebhookConfig{Name: e.ID})
				}
				// Also exercises the Uninstall path (DeleteWebhookEndpoints).
				Expect(c.DeleteWebhookEndpoints(configs)).To(BeNil())
			})
			Expect(err).To(BeNil())
			// One plain endpoint (<base>/root) + one Connect endpoint
			// (<base>/connect), both subscribed to balance.available.
			Expect(endpoints).To(HaveLen(2))

			var sawRoot, sawConnect bool
			for _, e := range endpoints {
				// ID -> PSPWebhookConfig.Name and the delete key (hard).
				// Secret -> signature verification (hard, create-only).
				// URL -> TrimPrefix + the root/connect routing (hard).
				// EnabledEvents -> stored in the config metadata.
				Expect(e.ID).ToNot(BeEmpty())
				Expect(e.Secret).ToNot(BeEmpty())
				Expect(strings.HasPrefix(e.Secret, "whsec_")).To(BeTrue(),
					"webhook secret %q does not look like a Stripe signing secret", e.Secret)
				Expect(strings.HasPrefix(e.URL, base)).To(BeTrue(),
					"webhook URL %q does not start with the requested base %q", e.URL, base)
				Expect(e.EnabledEvents).To(ContainElement(string(stripe.EventTypeBalanceAvailable)))
				switch {
				case strings.HasSuffix(e.URL, "/root"):
					sawRoot = true
				case strings.HasSuffix(e.URL, "/connect"):
					sawConnect = true
				}
			}
			Expect(sawRoot).To(BeTrue(), "no root webhook endpoint was created")
			Expect(sawConnect).To(BeTrue(), "no connect webhook endpoint was created")
		})
	})
})
