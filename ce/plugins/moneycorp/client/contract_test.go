//go:build contract

// Package client contract test for the Moneycorp connector.
//
// This is a CONTRACT test: it calls the real Moneycorp sandbox over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types) or in list ordering. It is gated behind the `contract`
// build tag so it never runs as part of `just tests` (which only enables
// `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	MONEYCORP_CONTRACT_CLIENT_ID=... MONEYCORP_CONTRACT_API_KEY=... \
//	    just contract-tests moneycorp
//
// Auth & endpoint: Moneycorp authenticates via POST /login with a JSON body
// {loginId, apiKey} (NOT the OAuth2 /oauth/token flow the stale api-reference
// doc shows) and then sends "Authorization: Bearer <accessToken>". New requires
// an explicit endpoint (there is no built-in host and Config.Endpoint is
// required), and sandbox/prod are different hosts, so the sandbox host is
// hardcoded to sandboxEndpoint (never risks pointing creds at prod) and only the
// two credentials are secrets. Without BOTH the suite Skips rather than fails.
//
// Dates: every timestamp the connector consumes is parsed with the layout
// moneycorpDateLayout ("2006-01-02T15:04:05.999999999"), NOT RFC3339 — there is
// no timezone. Amounts/balances are MAJOR-unit decimal strings carried as
// json.Number, so the smallest money-movement amount is "0.01" (not 1) and there
// is no overdraft flag — the money-movement specs source a funded account and
// Skip when none is funded.
//
// Everything is ACCOUNT-scoped (balances, recipients, transactions, transfers,
// payouts all hang off an account ID), so the suite discovers a usable account at
// runtime rather than hardcoding one.
//
// Ordering uses monotonic assertions (not pinned IDs): accounts come back id
// ascending (fetchNextAccounts derives its numeric LastIDCreated watermark from
// the last element), recipients createdAt ascending (fetchNextExternalAccounts
// watermark), and transactions createdAt ascending (fetchNextPayments watermark).
// The real dependency is the sort property, so there is nothing to bootstrap.
//
// Scope: all 7 client.Client methods are consumed by ingestion, so all are
// covered. Moneycorp exposes no webhook/create-recipient/reversal methods, so the
// only mutations are money movement (internal transfer + its GetTransfer
// read-back, and an outbound payout).
package client

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/contracttest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMoneycorpContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Moneycorp Contract Suite")
}

// sandboxEndpoint is Moneycorp's sandbox base URL (prod is corpapi.moneycorp.com).
// Hardcoded so demo credentials never risk hitting production; flip it if a
// tenant's sandbox lives on a different host.
const sandboxEndpoint = "https://sandbox-corpapi.moneycorp.com"

// moneycorpDateLayout is the timestamp layout every consumed Moneycorp date is
// parsed with (no timezone). Using time.RFC3339 here would be wrong.
const moneycorpDateLayout = "2006-01-02T15:04:05.999999999"

// contractPageSize mirrors the connector's PAGE_SIZE (Moneycorp's documented max
// is 10000). The collectAll* helpers page through the full list regardless.
const contractPageSize = 100

// contractMinAmount is the smallest money-movement amount as Moneycorp expects
// it: a MAJOR-unit decimal string. "0.01" is 1 minor unit for a 2-decimal
// currency (the sandbox house currencies — GBP/EUR/USD — are all 2-decimal).
const contractMinAmount = "0.01"

// contractMinFunded is the minimum parsed available balance (major units) an
// account must hold for us to safely source a contractMinAmount movement from it
// (Moneycorp has no overdraft flag).
const contractMinFunded = 1.0

// contractReference is the transfer/payout reference. Kept short and simple.
const contractReference = "Formance Contract Test"

// collectAllAccounts pages through every account in list order (id asc, the
// connector's sort). Pagination is 0-based to match fetchNextAccounts. Bounded to
// avoid an accidental unbounded loop.
func collectAllAccounts(ctx context.Context, c Client) ([]*Account, error) {
	var all []*Account
	for page := 0; page <= 5000; page++ {
		accounts, err := c.GetAccounts(ctx, page, contractPageSize)
		if err != nil {
			return nil, err
		}
		if len(accounts) == 0 {
			break
		}
		all = append(all, accounts...)
		if len(accounts) < contractPageSize {
			break
		}
	}
	return all, nil
}

// collectAllRecipients pages through every recipient of an account in list order
// (createdAt asc). Bounded to avoid an accidental unbounded loop.
func collectAllRecipients(ctx context.Context, c Client, accountID string) ([]*Recipient, error) {
	var all []*Recipient
	for page := 0; page <= 5000; page++ {
		recipients, err := c.GetRecipients(ctx, accountID, page, contractPageSize)
		if err != nil {
			return nil, err
		}
		if len(recipients) == 0 {
			break
		}
		all = append(all, recipients...)
		if len(recipients) < contractPageSize {
			break
		}
	}
	return all, nil
}

// collectAllTransactions walks an account's transactions the way fetchNextPayments
// does (zero lastCreatedAt = full backlog, createdAt asc). Bounded to avoid an
// accidental unbounded loop.
func collectAllTransactions(ctx context.Context, c Client, accountID string) ([]*Transaction, error) {
	var all []*Transaction
	for page := 0; page <= 5000; page++ {
		transactions, err := c.GetTransactions(ctx, accountID, page, contractPageSize, time.Time{})
		if err != nil {
			return nil, err
		}
		if len(transactions) == 0 {
			break
		}
		all = append(all, transactions...)
		if len(transactions) < contractPageSize {
			break
		}
	}
	return all, nil
}

// firstAccountWithBalances returns the first account that reports at least one
// balance, plus those balances. ok=false when no account has any balance.
func firstAccountWithBalances(ctx context.Context, c Client, accounts []*Account) (string, []*Balance, bool) {
	for _, a := range accounts {
		balances, err := c.GetAccountBalances(ctx, a.ID)
		Expect(err).To(BeNil())
		if len(balances) > 0 {
			return a.ID, balances, true
		}
	}
	return "", nil, false
}

// firstAccountWithRecipients returns the first account whose recipients endpoint
// SUCCEEDS and returns at least one recipient, plus those recipients in list
// order. Per-account errors are TOLERATED: the Moneycorp sandbox returns a 500
// ("Unexpected Exception") on GET /accounts/{id}/recipients for some accounts, so
// we skip a failing account and try the next rather than failing the spec on a
// sandbox-side error that is not schema drift. ok=false when no account yields
// recipients (the caller then Skips).
func firstAccountWithRecipients(ctx context.Context, c Client, accounts []*Account) (string, []*Recipient, bool) {
	for _, a := range accounts {
		recipients, err := collectAllRecipients(ctx, c, a.ID)
		if err != nil {
			continue
		}
		if len(recipients) > 0 {
			return a.ID, recipients, true
		}
	}
	return "", nil, false
}

// firstAccountWithTransactions returns the first account that has at least one
// transaction, plus its transactions in list order. ok=false when none has any.
func firstAccountWithTransactions(ctx context.Context, c Client, accounts []*Account) (string, []*Transaction, bool) {
	for _, a := range accounts {
		transactions, err := collectAllTransactions(ctx, c, a.ID)
		Expect(err).To(BeNil())
		if len(transactions) > 0 {
			return a.ID, transactions, true
		}
	}
	return "", nil, false
}

// fundedAccount is an account we can safely source a money movement from: one of
// its balances parses and holds at least contractMinFunded major units.
type fundedAccount struct {
	accountID string
	currency  string
}

// findFundedAccount returns the first account with an AvailableBalance that parses
// and is at least contractMinFunded, so a contractMinAmount (0.01) movement won't
// hit insufficient funds. ok=false when none is funded (the caller then Skips).
func findFundedAccount(ctx context.Context, c Client, accounts []*Account) (fundedAccount, bool) {
	for _, a := range accounts {
		balances, err := c.GetAccountBalances(ctx, a.ID)
		Expect(err).To(BeNil())
		for _, b := range balances {
			amount, ferr := b.Attributes.AvailableBalance.Float64()
			if ferr != nil {
				continue
			}
			if amount >= contractMinFunded {
				return fundedAccount{accountID: a.ID, currency: b.Attributes.CurrencyCode}, true
			}
		}
	}
	return fundedAccount{}, false
}

// payoutSource is a source account that is both funded and owns a recipient whose
// currency matches the funded balance — a payout must be sourced from an account
// that can fund it AND that the recipient belongs to (recipients are
// account-scoped and the payout endpoint is /accounts/{source}/payments).
type payoutSource struct {
	accountID   string
	recipientID string
	currency    string
}

// findPayoutSource scans accounts for one that (a) holds >= contractMinFunded in
// some currency and (b) has a recipient in that same currency. The recipients
// endpoint error is TOLERATED per account (sandbox 500s on some), so a failing
// account is skipped rather than failing the spec. ok=false when no such account
// exists (the caller then Skips).
func findPayoutSource(ctx context.Context, c Client, accounts []*Account) (payoutSource, bool) {
	for _, a := range accounts {
		balances, err := c.GetAccountBalances(ctx, a.ID)
		Expect(err).To(BeNil())
		funded := make(map[string]struct{})
		for _, b := range balances {
			amount, ferr := b.Attributes.AvailableBalance.Float64()
			if ferr == nil && amount >= contractMinFunded {
				funded[b.Attributes.CurrencyCode] = struct{}{}
			}
		}
		if len(funded) == 0 {
			continue
		}

		recipients, rerr := collectAllRecipients(ctx, c, a.ID)
		if rerr != nil {
			continue
		}
		for _, r := range recipients {
			if _, ok := funded[r.Attributes.BankAccountCurrency]; ok {
				return payoutSource{accountID: a.ID, recipientID: r.ID, currency: r.Attributes.BankAccountCurrency}, true
			}
		}
	}
	return payoutSource{}, false
}

var _ = Describe("Moneycorp API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		clientID := os.Getenv("MONEYCORP_CONTRACT_CLIENT_ID")
		apiKey := os.Getenv("MONEYCORP_CONTRACT_API_KEY")
		if clientID == "" || apiKey == "" {
			Skip("MONEYCORP_CONTRACT_CLIENT_ID and MONEYCORP_CONTRACT_API_KEY must be set to run the Moneycorp contract test")
		}

		// Bound each spec so a hung or slow sandbox call fails fast instead of
		// stalling the daily CI job indefinitely.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancel)
		// Hardcoded sandbox host: sandbox and prod are different hosts and the
		// sandbox credentials only authenticate against sandboxEndpoint.
		c = New("moneycorp", clientID, apiKey, sandboxEndpoint)
	})

	Describe("GetAccounts", func() {
		It("returns accounts whose shape matches what the connector consumes, id-ascending", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			ids := make([]int64, 0, len(accounts))
			for _, a := range accounts {
				// id is a HARD dependency: strconv.ParseInt(account.ID, 10, 64)
				// errors otherwise, and it is both the numeric LastIDCreated
				// watermark and the PSPAccount.Reference. accountName is
				// address-only (tolerated empty), so it is NOT asserted.
				Expect(a.ID).ToNot(BeEmpty())
				id, perr := strconv.ParseInt(a.ID, 10, 64)
				Expect(perr).To(BeNil(), "account id %q is not an int64", a.ID)
				ids = append(ids, id)
			}

			// The client requests sortBy=id.asc and derives LastIDCreated from the
			// last element, so ascending numeric id is the ordering contract.
			contracttest.AssertNonDecreasing(ids, "account id")
		})
	})

	Describe("GetAccountBalances", func() {
		It("returns balances whose shape matches what the connector consumes", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			_, balances, ok := firstAccountWithBalances(ctx, c, accounts)
			if !ok {
				Skip("no account with balances in the sandbox to exercise GetAccountBalances")
			}

			for _, b := range balances {
				// currencyCode is a HARD dependency: currency.GetPrecision errors
				// on an unknown/empty currency. availableBalance is the only
				// balance field consumed (parsed as a major-unit decimal via
				// GetAmountWithPrecisionFromString); the other balances and the
				// balance id are NOT consumed.
				Expect(b.Attributes.CurrencyCode).ToNot(BeEmpty())
				contracttest.AssertDecimalAmount(b.Attributes.AvailableBalance, "balance availableBalance")
			}
		})
	})

	Describe("GetRecipients", func() {
		It("returns recipients whose shape matches what the connector consumes, createdAt-ascending", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			_, recipients, ok := firstAccountWithRecipients(ctx, c, accounts)
			if !ok {
				Skip("no account with recipients in the sandbox to exercise GetRecipients")
			}

			createdAts := make([]int64, 0, len(recipients))
			for _, r := range recipients {
				// id -> PSPAccount.Reference (hard). createdAt is hard:
				// recipientToPSPAccounts parses it with moneycorpDateLayout and
				// errors otherwise; it also drives the LastCreatedAt watermark.
				// bankAccountName and bankAccountCurrency are address-only /
				// FormatAsset (tolerated), so they are NOT asserted.
				Expect(r.ID).ToNot(BeEmpty())
				t, perr := time.Parse(moneycorpDateLayout, r.Attributes.CreatedAt)
				Expect(perr).To(BeNil(), "recipient createdAt %q is not %s", r.Attributes.CreatedAt, moneycorpDateLayout)
				createdAts = append(createdAts, t.UnixNano())
			}

			// The client requests sortBy=createdAt.asc and derives its watermark
			// from the last element, so ascending createdAt is the ordering contract.
			contracttest.AssertNonDecreasing(createdAts, "recipient createdAt")
		})
	})

	Describe("GetTransactions", func() {
		It("returns transactions whose shape matches what the connector consumes, createdAt-ascending", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			_, transactions, ok := firstAccountWithTransactions(ctx, c, accounts)
			if !ok {
				Skip("no account with transactions in the sandbox to exercise GetTransactions")
			}

			createdAts := make([]int64, 0, len(transactions))
			for _, tx := range transactions {
				// id -> reference (hard). createdAt is hard: toPSPPayments parses
				// it with moneycorpDateLayout (errors otherwise) and drives the
				// LastCreatedAt watermark. transactionCurrency -> GetPrecision
				// (hard). transactionAmount -> decimal (hard). transactionDirection
				// and transactionType drive the type/source-dest switch (hard).
				// accountId -> source/dest reference + GetTransfer key (hard).
				Expect(tx.ID).ToNot(BeEmpty())
				Expect(tx.Attributes.Currency).ToNot(BeEmpty())
				Expect(tx.Attributes.Direction).ToNot(BeEmpty())
				Expect(tx.Attributes.Type).ToNot(BeEmpty())
				Expect(tx.Attributes.AccountID).To(BeNumerically(">", 0))
				contracttest.AssertDecimalAmount(tx.Attributes.Amount, "transaction transactionAmount")
				t, perr := time.Parse(moneycorpDateLayout, tx.Attributes.CreatedAt)
				Expect(perr).To(BeNil(), "transaction createdAt %q is not %s", tx.Attributes.CreatedAt, moneycorpDateLayout)
				createdAts = append(createdAts, t.UnixNano())

				// For Transfer/Payment rows the related object id is used as the
				// GetTransfer key / the payment reference, so it must be present.
				if tx.Attributes.Type == "Transfer" || tx.Attributes.Type == "Payment" {
					Expect(tx.Relationships.Data.ID).ToNot(BeEmpty(), "relationships.data.id empty for a %s transaction", tx.Attributes.Type)
				}
			}

			// The client requests sortBy=createdAt.asc and derives LastCreatedAt
			// from the last element, so ascending createdAt is the ordering contract.
			contracttest.AssertNonDecreasing(createdAts, "transaction createdAt")
		})
	})

	Describe("InitiateTransfer", func() {
		// Internal transfer between two of our own accounts: money stays on the
		// platform. Moneycorp has no overdraft flag, so we source from a funded
		// account and Skip when none is funded. Smallest amount, unique UUID
		// idempotency key per run. Then GetTransfer(sourceAccountID, id) exercises
		// the ingestion get-by-id path (fetchAndTranslateTransfer).
		It("initiates a minimal internal transfer and reads it back", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())
			if len(accounts) < 2 {
				Skip("need at least 2 accounts in the sandbox to exercise a transfer")
			}

			funded, ok := findFundedAccount(ctx, c, accounts)
			if !ok {
				Skip("no funded account to source a transfer (Moneycorp has no overdraft)")
			}

			// Destination = any account other than the funded source.
			var destination string
			for _, a := range accounts {
				if a.ID != funded.accountID {
					destination = a.ID
					break
				}
			}
			if destination == "" {
				Skip("no distinct destination account to exercise a transfer")
			}

			resp, err := c.InitiateTransfer(ctx, &TransferRequest{
				IdempotencyKey:     contracttest.UUIDRef(),
				SourceAccountID:    funded.accountID,
				ReceivingAccountID: destination,
				TransferAmount:     contractMinAmount,
				TransferCurrency:   funded.currency,
				TransferReference:  contractReference,
				ClientReference:    contractReference,
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Attributes.TransferCurrency).ToNot(BeEmpty())
			Expect(resp.Attributes.TransferStatus).ToNot(BeEmpty())
			Expect(resp.Attributes.SendingAccountID).To(BeNumerically(">", 0))
			contracttest.AssertDecimalAmount(resp.Attributes.TransferAmount, "transfer transferAmount")
			_, perr := time.Parse(moneycorpDateLayout, resp.Attributes.CreatedAt)
			Expect(perr).To(BeNil(), "transfer createdAt %q is not %s", resp.Attributes.CreatedAt, moneycorpDateLayout)

			// get-by-id: the same path ingestion uses to enrich a Transfer/Debit.
			got, err := c.GetTransfer(ctx, funded.accountID, resp.ID)
			Expect(err).To(BeNil())
			Expect(got).ToNot(BeNil())
			Expect(got.ID).ToNot(BeEmpty())
			Expect(got.Attributes.TransferCurrency).ToNot(BeEmpty())
			Expect(got.Attributes.TransferStatus).ToNot(BeEmpty())
			contracttest.AssertDecimalAmount(got.Attributes.TransferAmount, "fetched transfer transferAmount")
			_, perr = time.Parse(moneycorpDateLayout, got.Attributes.CreatedAt)
			Expect(perr).To(BeNil(), "fetched transfer createdAt %q is not %s", got.Attributes.CreatedAt, moneycorpDateLayout)
		})
	})

	Describe("InitiatePayout", func() {
		// Outbound payout to a recipient. Source = a funded account; the recipient
		// must use the funded currency (the payment must be both funded AND
		// accepted). Moneycorp has no overdraft flag, so Skip when nothing is
		// funded or no currency-matched recipient exists. Fields mirror
		// createPayout (PaymentMethod "Standard"; PaymentDate/PaymentPurpose left
		// empty as the connector sends them). Smallest amount, unique UUID key.
		It("initiates a minimal outbound payout to a currency-matched recipient", func() {
			accounts, err := collectAllAccounts(ctx, c)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			src, ok := findPayoutSource(ctx, c, accounts)
			if !ok {
				Skip("no funded account with a same-currency recipient to exercise a payout (Moneycorp has no overdraft; sandbox may 500 on recipients)")
			}

			resp, err := c.InitiatePayout(ctx, &PayoutRequest{
				IdempotencyKey:   contracttest.UUIDRef(),
				SourceAccountID:  src.accountID,
				RecipientID:      src.recipientID,
				PaymentAmount:    contractMinAmount,
				PaymentCurrency:  src.currency,
				PaymentMethod:    "Standard",
				PaymentReference: contractReference,
				ClientReference:  contractReference,
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Attributes.PaymentCurrency).ToNot(BeEmpty())
			Expect(resp.Attributes.PaymentStatus).ToNot(BeEmpty())
			Expect(resp.Attributes.AccountID).To(BeNumerically(">", 0))
			contracttest.AssertDecimalAmount(resp.Attributes.PaymentAmount, "payout paymentAmount")
			_, perr := time.Parse(moneycorpDateLayout, resp.Attributes.CreatedAt)
			Expect(perr).To(BeNil(), "payout createdAt %q is not %s", resp.Attributes.CreatedAt, moneycorpDateLayout)
		})
	})
})
