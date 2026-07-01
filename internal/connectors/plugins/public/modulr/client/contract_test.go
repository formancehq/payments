//go:build contract

// Package client contract test for the Modulr connector.
//
// This is a CONTRACT test: it calls the real Modulr sandbox over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types) or in list ordering. It is gated behind the `contract`
// build tag so it never runs as part of `just tests` (which only enables
// `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	MODULR_CONTRACT_API_KEY=... just contract-tests modulr
//
// Auth & endpoint: Modulr's sandbox offers a TOKEN auth mode on the
// api-sandbox-token host where the API Key is passed directly as the
// Authorization header (no HMAC signing). The connector targets exactly that:
// New defaults the endpoint to SandboxAPIEndpoint and RoundTrip sends only
// "Authorization: <apiKey>". So the endpoint is hardcoded to SandboxAPIEndpoint
// (never risks pointing at prod) and only the API key is a secret. New also
// requires a non-empty apiSecret, but the token endpoint discards it (the HMAC
// headers New computes are never applied), so it is a non-secret test constant.
// Without MODULR_CONTRACT_API_KEY the suite Skips rather than fails.
//
// Dates: every timestamp the connector consumes is parsed with the layout
// modulrDateLayout ("2006-01-02T15:04:05.999-0700", e.g. 2026-06-18T13:54:29+0000),
// NOT RFC3339. Amounts/balances are MAJOR-unit decimal strings, so the smallest
// money-movement amount is "0.01" (not 1) and there is no overdraft flag — the
// money-movement specs source a funded account and Skip when none is funded.
//
// Ordering uses monotonic-date assertions (not pinned IDs): accounts come back
// createdDate-ascending (fetchNextAccounts derives its watermark from the last
// element), beneficiaries created-ascending (same watermark pattern), and
// transactions newest-first (the fetchNextPayments drain window freezes its
// ceiling from page[0] and reverses each page). The real dependency is the sort
// property, so there is nothing to bootstrap.
//
// Scope: only methods ingestion actually consumes. GetPayments and GetPayout
// exist on the client but are UNUSED by any fill*/fetch* path, so they are out
// of scope (covered by unit tests). Modulr exposes no webhook/create-beneficiary
// methods, so the only mutations are money movement.
package client

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/contracttest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestModulrContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Modulr Contract Suite")
}

// modulrDateLayout is the timestamp layout every consumed Modulr date is parsed
// with (numeric offset without colon, optional millis). Using time.RFC3339 here
// would be wrong.
const modulrDateLayout = "2006-01-02T15:04:05.999-0700"

// contractPageSize is the connector's PAGE_SIZE (Modulr's max is 500).
const contractPageSize = 100

// contractMinAmount is the smallest money-movement amount as Modulr expects it:
// a MAJOR-unit decimal string. "0.01" is 1 minor unit for a 2-decimal currency
// (all Modulr-supported currencies except JPY are 2-decimal).
const contractMinAmount = "0.01"

// contractMinAmountFloat is contractMinAmount as a float, for the funded-account
// balance comparison.
const contractMinAmountFloat = 0.01

// contractReference is the payment reference/externalReference. Kept short and
// alphanumeric to satisfy Modulr's reference constraints (<=18 chars, [A-Za-z0-9 ]).
const contractReference = "Formance CT"

// contractUnusedSecret is passed to New as apiSecret. Token auth (the sandbox
// endpoint the connector targets) ignores it — it only needs to be non-empty to
// satisfy New's credential guard — so it is a non-secret test constant.
const contractUnusedSecret = "contract-test-unused-secret"

// contractPayoutSourceAccountID / contractPayoutBeneficiaryID pin a known-good,
// same-currency DOMESTIC (source account, beneficiary) pair for the outbound
// payout spec. Modulr classifies a currency/country-mismatched payout as an
// invalid international payment, and the client cannot pre-check a beneficiary's
// currency (the Beneficiary struct exposes none), so the payout can only be
// reliably exercised against a pinned pair. They start empty → the payout spec
// Skips until filled from the sandbox.
const (
	contractPayoutSourceAccountID = "A1216EG2"
	// Beneficiary for the 00-00-00 / 31926819 destination (same currency as the
	// source account) — a sort code/account can't be used directly, since the
	// payout Destination is a beneficiary reference.
	contractPayoutBeneficiaryID = "B21008BDB6"
)

// firstAccountWithTransactions returns the first account that has at least one
// transaction, plus its page-0 transactions (newest-first) and the reported
// totalPages, so the transaction schema/ordering specs run against real data.
// ok=false when no account has any transactions.
func firstAccountWithTransactions(ctx context.Context, c Client, accounts []Account) (string, []Transaction, int, bool) {
	for _, a := range accounts {
		txns, totalPages, err := c.GetTransactions(ctx, a.ID, 0, contractPageSize, time.Time{}, time.Time{})
		Expect(err).To(BeNil())
		if len(txns) > 0 {
			return a.ID, txns, totalPages, true
		}
	}
	return "", nil, 0, false
}

// findFundedSourceAndDest returns a funded source account and a DISTINCT
// destination account of the SAME currency (Modulr accounts are single-currency,
// so an internal transfer needs both ends in one currency). ok=false when no
// such pair exists.
func findFundedSourceAndDest(ctx context.Context, c Client, accounts []Account, minAmount float64) (source, dest, currency string, ok bool) {
	for _, a := range accounts {
		acct, err := c.GetAccount(ctx, a.ID)
		Expect(err).To(BeNil())
		bal, perr := strconv.ParseFloat(acct.Balance, 64)
		if perr != nil || bal < minAmount {
			continue
		}
		for _, b := range accounts {
			if b.ID != a.ID && b.Currency == acct.Currency {
				return a.ID, b.ID, acct.Currency, true
			}
		}
	}
	return "", "", "", false
}

// assertTransactionShape validates the fields the connector hard-depends on for
// every transaction in the batch. TransactionDate is parsed for every row (the
// drain window's ceiling + per-row filter). PostedDate is parsed only in the
// non-transfer path, so it is asserted only when present (a transfer row may omit
// it). id / credit / description / additionalInfo are not the reference / bool /
// metadata, so they are NOT asserted.
func assertTransactionShape(txns []Transaction) {
	for _, tx := range txns {
		// source_id -> Reference (hard) AND the GetTransfer key for transfers.
		Expect(tx.SourceID).ToNot(BeEmpty())
		// type -> matchTransactionType routes payin/payout/transfer (hard).
		Expect(tx.Type).ToNot(BeEmpty())
		// account.currency -> precision map key + FormatAsset; drift here would
		// silently drop every transaction, so its presence is a real contract.
		Expect(tx.Account.Currency).ToNot(BeEmpty())
		// amount is a major-unit decimal string parsed via GetAmountWithPrecisionFromString.
		contracttest.AssertDecimalAmount(tx.Amount, "transaction amount")
		_, perr := time.Parse(modulrDateLayout, tx.TransactionDate)
		Expect(perr).To(BeNil(), "transaction transactionDate %q is not %s", tx.TransactionDate, modulrDateLayout)
		if tx.PostedDate != "" {
			_, perr := time.Parse(modulrDateLayout, tx.PostedDate)
			Expect(perr).To(BeNil(), "transaction postedDate %q is not %s", tx.PostedDate, modulrDateLayout)
		}
	}
}

var _ = Describe("Modulr API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		apiKey := os.Getenv("MODULR_CONTRACT_API_KEY")
		if apiKey == "" {
			Skip("MODULR_CONTRACT_API_KEY must be set to run the Modulr contract test")
		}

		ctx = context.Background()
		var err error
		c, err = New("modulr", apiKey, contractUnusedSecret, SandboxAPIEndpoint)
		Expect(err).To(BeNil())
	})

	Describe("GetAccounts", func() {
		It("returns accounts whose shape matches what the connector consumes, createdDate-ascending", func() {
			accounts, err := c.GetAccounts(ctx, 0, contractPageSize, time.Time{})
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			createdAts := make([]int64, 0, len(accounts))
			for _, a := range accounts {
				// id -> Reference (hard). createdDate is hard: fillAccounts parses
				// it and errors otherwise. currency drives DefaultAsset (FormatAsset)
				// and is reused by balances. name is address-only (tolerated empty);
				// status/customerId/identifiers/directDebit are metadata/Raw only.
				Expect(a.ID).ToNot(BeEmpty())
				Expect(a.Currency).ToNot(BeEmpty())
				t, perr := time.Parse(modulrDateLayout, a.CreatedDate)
				Expect(perr).To(BeNil(), "account createdDate %q is not %s", a.CreatedDate, modulrDateLayout)
				createdAts = append(createdAts, t.UnixNano())
			}

			// The client requests sortField=createdDate&sortOrder=asc and derives
			// its watermark from the last element, so ascending createdDate is the
			// ordering contract.
			contracttest.AssertNonDecreasing(createdAts, "account createdDate")
		})
	})

	Describe("GetAccount", func() {
		It("returns an account by ID with a decimal balance", func() {
			accounts, err := c.GetAccounts(ctx, 0, contractPageSize, time.Time{})
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			acct, err := c.GetAccount(ctx, accounts[0].ID)
			Expect(err).To(BeNil())
			Expect(acct).ToNot(BeNil())

			// balance is hard: fetchNextBalances parses it via
			// GetAmountWithPrecisionFromString (a major-unit decimal string).
			// currency is the precision map key. id keeps parity with the list.
			Expect(acct.ID).ToNot(BeEmpty())
			Expect(acct.Currency).ToNot(BeEmpty())
			contracttest.AssertDecimalAmount(json.Number(acct.Balance), "account balance")
		})
	})

	Describe("GetBeneficiaries", func() {
		It("returns beneficiaries whose shape matches what the connector consumes, created-ascending", func() {
			beneficiaries, err := c.GetBeneficiaries(ctx, 0, contractPageSize, time.Time{})
			Expect(err).To(BeNil())

			createdAts := make([]int64, 0, len(beneficiaries))
			for _, b := range beneficiaries {
				// id -> Reference (hard). created is hard: fillBeneficiaries parses
				// it and errors otherwise. name is address-only (tolerated empty).
				Expect(b.ID).ToNot(BeEmpty())
				t, perr := time.Parse(modulrDateLayout, b.Created)
				Expect(perr).To(BeNil(), "beneficiary created %q is not %s", b.Created, modulrDateLayout)
				createdAts = append(createdAts, t.UnixNano())
			}

			// fetchNextExternalAccounts derives its watermark from the last
			// element, assuming ascending created (Modulr sends no sort param).
			// Asserting it validates the connector's own assumption.
			contracttest.AssertNonDecreasing(createdAts, "beneficiary created")
		})
	})

	Describe("GetTransactions", func() {
		It("returns transactions whose shape matches what the connector consumes, newest-first", func() {
			accounts, err := c.GetAccounts(ctx, 0, contractPageSize, time.Time{})
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			_, txns, totalPages, ok := firstAccountWithTransactions(ctx, c, accounts)
			if !ok {
				Skip("no account with transactions in the sandbox to exercise GetTransactions")
			}

			// fetchNextPayments reads totalPages to find the oldest page.
			Expect(totalPages).To(BeNumerically(">=", 1))
			assertTransactionShape(txns)

			// The drain window hard-depends on newest-first ordering (it freezes
			// the ceiling from page[0] and reverses each page). Reverse to
			// oldest-first and assert non-decreasing transactionDate.
			transactionDates := make([]int64, 0, len(txns))
			for i := len(txns) - 1; i >= 0; i-- {
				t, perr := time.Parse(modulrDateLayout, txns[i].TransactionDate)
				Expect(perr).To(BeNil())
				transactionDates = append(transactionDates, t.UnixNano())
			}
			contracttest.AssertNonDecreasing(transactionDates, "transaction transactionDate (reversed to oldest-first)")
		})
	})

	Describe("InitiateTransfer", func() {
		// Internal transfer between two of our own same-currency accounts: money
		// stays on the platform. Modulr has no overdraft, so we source from a
		// funded account and Skip when none is funded. Smallest amount, unique
		// UUID nonce per run. Then GetTransfer(id) exercises the ingestion
		// get-by-id path (fetchAndTranslateTransfer).
		It("initiates a minimal internal transfer and reads it back", func() {
			accounts, err := c.GetAccounts(ctx, 0, contractPageSize, time.Time{})
			Expect(err).To(BeNil())
			if len(accounts) < 2 {
				Skip("need at least 2 accounts in the sandbox to exercise a transfer")
			}

			source, dest, curr, ok := findFundedSourceAndDest(ctx, c, accounts, contractMinAmountFloat)
			if !ok {
				Skip("no funded account with a same-currency destination (Modulr has no overdraft)")
			}

			resp, err := c.InitiateTransfer(ctx, &TransferRequest{
				IdempotencyKey:  contracttest.UUIDRef(),
				SourceAccountID: source,
				Destination: Destination{
					Type: string(DestinationTypeAccount),
					ID:   dest,
				},
				Currency:          curr,
				Amount:            json.Number(contractMinAmount),
				Reference:         contractReference,
				ExternalReference: contractReference,
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
			_, perr := time.Parse(modulrDateLayout, resp.CreatedDate)
			Expect(perr).To(BeNil(), "transfer createdDate %q is not %s", resp.CreatedDate, modulrDateLayout)

			// get-by-id: the same path ingestion uses to enrich a transfer.
			got, err := c.GetTransfer(ctx, resp.ID)
			Expect(err).To(BeNil())
			Expect(got.ID).ToNot(BeEmpty())
			Expect(got.Status).ToNot(BeEmpty())
			_, perr = time.Parse(modulrDateLayout, got.CreatedDate)
			Expect(perr).To(BeNil(), "fetched transfer createdDate %q is not %s", got.CreatedDate, modulrDateLayout)
		})
	})

	Describe("InitiatePayout", func() {
		// Outbound payout to a beneficiary. The Beneficiary struct exposes no
		// currency, so a runtime-picked (source, beneficiary) pair can be routed
		// as an invalid international payment. We therefore pin a known-good,
		// same-currency DOMESTIC pair; the currency is derived from the pinned
		// source account. Skips only if the pair is unpinned or the source is
		// underfunded (Modulr has no overdraft) — any other rejection is real
		// drift and fails. Smallest amount, unique UUID nonce per run.
		It("initiates a minimal outbound payout to the pinned beneficiary", func() {
			if contractPayoutSourceAccountID == "" || contractPayoutBeneficiaryID == "" {
				Skip("contractPayoutSourceAccountID/contractPayoutBeneficiaryID are not populated — pin a known-good same-currency domestic (source account, beneficiary) pair from the sandbox to enable the payout contract")
			}

			acct, err := c.GetAccount(ctx, contractPayoutSourceAccountID)
			Expect(err).To(BeNil())
			bal, perr := strconv.ParseFloat(acct.Balance, 64)
			Expect(perr).To(BeNil(), "pinned source account balance %q is not decimal", acct.Balance)
			if bal < contractMinAmountFloat {
				Skip("pinned payout source account is underfunded (Modulr has no overdraft)")
			}

			resp, err := c.InitiatePayout(ctx, &PayoutRequest{
				IdempotencyKey:  contracttest.UUIDRef(),
				SourceAccountID: contractPayoutSourceAccountID,
				Destination: Destination{
					Type: string(DestinationTypeBeneficiary),
					ID:   contractPayoutBeneficiaryID,
				},
				// Domestic payout: use the source account's own currency.
				Currency:          acct.Currency,
				Amount:            json.Number(contractMinAmount),
				Reference:         contractReference,
				ExternalReference: contractReference,
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
			_, perr = time.Parse(modulrDateLayout, resp.CreatedDate)
			Expect(perr).To(BeNil(), "payout createdDate %q is not %s", resp.CreatedDate, modulrDateLayout)
		})
	})
})
