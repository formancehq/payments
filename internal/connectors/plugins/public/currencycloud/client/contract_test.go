//go:build contract

// Package client contract test for the CurrencyCloud connector.
//
// This is a CONTRACT test: it calls the real CurrencyCloud demo/sandbox
// environment over the network through the same client.Client the connector
// uses, and asserts that the responses the Payments project depends on have not
// drifted in schema (field presence + types) or in list ordering. It is gated
// behind the `contract` build tag so it never runs as part of `just tests`
// (which only enables `-tags it`); it runs daily via the contract-tests GitHub
// workflow.
//
// Run locally:
//
//	CURRENCYCLOUD_CONTRACT_LOGIN_ID=... CURRENCYCLOUD_CONTRACT_API_KEY=... \
//	    just contract-tests currencycloud
//
// The connector targets CurrencyCloud's v2 API. Demo/sandbox and production are
// DIFFERENT hosts (devapi.currencycloud.com vs api.currencycloud.com) and
// credentials are environment-specific, so the endpoint is hardcoded to the
// demo host (DevAPIEndpoint) and only the login_id/api_key are secrets. Without
// both env vars the suite Skips rather than fails, so it is safe to run anywhere.
//
// Money movement: the InitiateTransfer / InitiatePayout specs make REAL calls
// against the demo environment at the smallest possible amount (0.01) with a
// unique idempotency Reference (unique_request_id) per run. CurrencyCloud has NO
// AllowOverdraft, so these specs derive the source account + currency from a
// GetBalances entry that is actually funded and Skip when the demo has no funded
// account (or too few accounts/beneficiaries). They accumulate demo state by
// design (accepted).
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

func TestCurrencyCloudContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CurrencyCloud Contract Suite")
}

// contractPageSize is CurrencyCloud's maximum per_page for the find/list APIs.
// The API rejects anything larger ("per_page can not be bigger than 25"), which
// is why the connector's PAGE_SIZE is also 25. The collectAll* helpers page
// through the full list, so this cap only affects how many round-trips we make.
const contractPageSize = 25

// contractMinAmount is the smallest amount the money-movement specs send. It is
// a major-unit decimal string valid for the 2-decimal currencies we restrict
// funding selection to. contractMinFunded is the minimum parsed balance (major
// units) an account must hold for us to source a 0.01 movement from it.
const (
	contractMinAmount = "0.01"
	contractMinFunded = 1.0
)

// expectedAccountIDs / expectedBeneficiaryIDs pin the known, seeded order of the
// demo environment's accounts and beneficiaries. They start empty; the schema
// specs print the live IDs to stderr as a paste-ready Go literal (bootstrap log)
// so a maintainer can fill these in to enable the ordering contract.
//
// Both use a growth/mutation-tolerant SUBSEQUENCE match (contracttest.FilterToPinned),
// not exact equality:
//   - accounts are ordered by updated_at asc; a money-movement run can bump an
//     account's updated_at and reorder the list. If that makes the pin flap,
//     leave expectedAccountIDs empty (the ordering spec then Skips).
//   - beneficiaries are ordered by created_at asc (immutable); new/out-of-band
//     beneficiaries append at the end and do not disturb the pinned order.
var (
	expectedAccountIDs = []string{
		"13b643d7-2e61-4804-9119-3f68ff98fadd",
		"c8e2ea2b-d54c-412c-abb1-a8b209abf78c",
		"d2e59ef9-00af-4cee-9d31-e00422a53148",
		"8d5072c6-cf68-48de-98e6-ea1ec81abdf9",
	}
	expectedBeneficiaryIDs = []string{
		"c7d10d60-163a-4fa5-9975-64a262cc5472",
		"f97e4501-c1b2-44b4-98f4-05d70b033f97",
	}
)

// expectedTransactionIDs pins the known transactions in the chronological order
// the connector consumes them (updated_at asc). Starts empty; set
// CURRENCYCLOUD_CONTRACT_BOOTSTRAP=1 and fill it from the bootstrap log to enable
// the ordering contract. Uses a growth/mutation-tolerant SUBSEQUENCE match: the
// money-movement specs create new transactions each run (appended at the tail of
// the updated_at asc walk), so exact equality can't hold. Pin OLD, already-settled
// transactions whose updated_at no longer changes — a pending transaction that
// later settles would bump its updated_at and reorder, so do NOT pin those.
var expectedTransactionIDs = []string{
	"8a06f27a-e264-49f6-93e0-793a9274966e",
	"6c77f0bf-d92e-48cf-9a51-d09cfbc2d18c",
	"6c77f0bf-d92e-48cf-9a51-d09cfbc2d18d",
	"0a8ee85d-f14f-4d62-a631-562f2de3743c",
	"82ba2d34-8a57-4941-836f-307d4b156471",
	"96233fdd-dccb-4b7e-a061-c586a4789218",
	"bff8139f-dac4-4610-9e02-eb9d73999157",
	"c1dad133-f873-4598-8c7b-1ffb441064c1",
	"c4daf056-aa94-4497-93e5-a9d9584346b8",
	"1bc5c09d-41de-4686-b296-7d5fe52bb8c6",
	"1e61dc50-8845-4001-bfde-9c81b2952a7a",
	"268c12a7-04f4-44db-bb5c-77a326d470a1",
	"4e128aff-6cad-4939-8ff3-24f353dc1dd4",
	"5959c81c-67bb-4c68-8b41-3b6a586fc77e",
	"8c1b2fc2-ada2-470f-a663-c9558d83fa47",
	"9be6840c-1745-44d6-9503-d167bfa76775",
	"da747db7-2f0a-4d52-8def-bfbb3bf53426",
	"5c438d17-8578-40e3-96cf-b87b68e61848",
	"028b3e6b-9ff8-4a08-8a2b-b7299b8a249a",
	"54da3596-0705-40dd-8d43-902ab4f9c104",
	"cb152e5a-a5f1-46ba-af96-61ce401f3e29",
	"d3bcc40f-1d72-4c28-b8c5-6446f8c20777",
	"d62f29ff-1340-419c-9505-3464c9a4d841",
	"75f4a8de-f775-49b4-a299-2570096c82c0",
	"ecf1b6ef-298b-497f-b666-ce5b5845e65d",
	"57ad7754-7693-4979-8371-446d8184888e",
	"1bfb607f-1fbf-4ea4-89c4-aa0ece714382",
	"53c52549-67c8-4d73-bb7c-658c56a0b127",
	"6e44ca23-5d47-475d-abab-9fd77890c8d4",
	"9ca9ce54-a1b2-4fc9-aa90-efb3358857f4",
	"14fb6763-7460-49ac-9d14-fb1eff5329ba",
	"b716d91f-b913-4a8f-8f72-27a82db8d962",
	"e2b227fe-b857-458e-be86-29435c871ad5",
	"bc77b679-9b05-419a-ab01-f22271eb6170",
}

// collectAllAccountIDs pages through every account and returns their IDs in list
// order (updated_at asc). Bounded to avoid an accidental unbounded loop.
func collectAllAccountIDs(ctx context.Context, c Client) ([]string, error) {
	var ids []string
	for page := 1; page <= 5000; page++ {
		accounts, nextPage, err := c.GetAccounts(ctx, page, contractPageSize)
		if err != nil {
			return nil, err
		}
		if len(accounts) == 0 {
			break
		}
		for _, a := range accounts {
			ids = append(ids, a.ID)
		}
		if nextPage == -1 {
			break
		}
	}
	return ids, nil
}

// collectAllBeneficiaryIDs pages through every beneficiary and returns their IDs
// in list order (created_at asc). Bounded to avoid an accidental unbounded loop.
func collectAllBeneficiaryIDs(ctx context.Context, c Client) ([]string, error) {
	var ids []string
	for page := 1; page <= 5000; page++ {
		beneficiaries, nextPage, err := c.GetBeneficiaries(ctx, page, contractPageSize)
		if err != nil {
			return nil, err
		}
		if len(beneficiaries) == 0 {
			break
		}
		for _, b := range beneficiaries {
			ids = append(ids, b.ID)
		}
		if nextPage == -1 {
			break
		}
	}
	return ids, nil
}

// collectAllTransactionIDs walks the transactions list the same way the connector
// does (a full backlog read: zero updatedAtFrom, updated_at asc) and returns every
// transaction ID in that chronological order. Bounded to avoid an accidental
// unbounded loop.
func collectAllTransactionIDs(ctx context.Context, c Client) ([]string, error) {
	var ids []string
	for page := 1; page <= 5000; page++ {
		transactions, nextPage, err := c.GetTransactions(ctx, page, contractPageSize, time.Time{})
		if err != nil {
			return nil, err
		}
		if len(transactions) == 0 {
			break
		}
		for _, tx := range transactions {
			ids = append(ids, tx.ID)
		}
		if nextPage == -1 {
			break
		}
	}
	return ids, nil
}

// fundedBalance is a demo balance we can safely source a money movement from: it
// parses and holds at least contractMinFunded major units.
type fundedBalance struct {
	accountID string
	currency  string
}

// findFundedBalance returns the first balance whose amount parses and is at least
// contractMinFunded, so a contractMinAmount (0.01) movement won't hit NSF. Returns
// ok=false when no such balance exists (the caller then Skips).
func findFundedBalance(balances []*Balance) (fundedBalance, bool) {
	for _, b := range balances {
		amount, err := b.Amount.Float64()
		if err != nil {
			continue
		}
		if amount >= contractMinFunded {
			return fundedBalance{accountID: b.AccountID, currency: b.Currency}, true
		}
	}
	return fundedBalance{}, false
}

var _ = Describe("CurrencyCloud API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		loginID := os.Getenv("CURRENCYCLOUD_CONTRACT_LOGIN_ID")
		apiKey := os.Getenv("CURRENCYCLOUD_CONTRACT_API_KEY")
		if loginID == "" || apiKey == "" {
			Skip("CURRENCYCLOUD_CONTRACT_LOGIN_ID and CURRENCYCLOUD_CONTRACT_API_KEY must be set to run the CurrencyCloud contract test")
		}

		ctx = context.Background()
		// Hardcoded demo host: sandbox and prod are different hosts and the demo
		// credentials only authenticate against DevAPIEndpoint.
		c = New("currencycloud", loginID, apiKey, DevAPIEndpoint)
	})

	Describe("GetAccounts", func() {
		It("returns accounts whose shape matches what the connector consumes", func() {
			accounts, _, err := c.GetAccounts(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			for _, a := range accounts {
				// ID is the PSPAccount.Reference (hard dependency).
				Expect(a.ID).ToNot(BeEmpty())
				// created_at is a hard dependency: it drives the watermark
				// (fillAccounts compares it) and a malformed value fails JSON
				// unmarshal at the client layer. account_name is tolerated empty
				// (the connector only takes its address), so it is NOT asserted.
				Expect(a.CreatedAt.IsZero()).To(BeFalse(), "account created_at is zero/unset")
			}

			if contracttest.BootstrapEnabled("CURRENCYCLOUD") {
				allIDs, err := collectAllAccountIDs(ctx, c)
				Expect(err).To(BeNil())
				contracttest.LogBootstrap("expectedAccountIDs", allIDs)
			}
		})

		It("keeps the known accounts in their expected, stable relative order", func() {
			if len(expectedAccountIDs) == 0 {
				Skip("expectedAccountIDs is not populated — set CURRENCYCLOUD_CONTRACT_BOOTSTRAP=1 and fill it from the bootstrap log to enable the ordering contract")
			}

			// Accounts are ordered updated_at asc and the money-movement specs can
			// bump an account's updated_at, so assert only that the *pinned*
			// accounts retain their relative order (ignoring any others), walking
			// all pages so pinned IDs are still found once they spill past page 1.
			allIDs, err := collectAllAccountIDs(ctx, c)
			Expect(err).To(BeNil())
			gotKnownIDs := contracttest.FilterToPinned(allIDs, expectedAccountIDs)
			Expect(gotKnownIDs).To(Equal(expectedAccountIDs))
		})
	})

	Describe("GetBalances", func() {
		It("returns balances whose shape matches what the connector consumes", func() {
			balances, _, err := c.GetBalances(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())

			for _, b := range balances {
				// account_id, currency and amount are all hard dependencies:
				// account_id -> AccountReference, currency -> supported-currency
				// map key, amount -> parsed via GetAmountWithPrecisionFromString.
				Expect(b.AccountID).ToNot(BeEmpty())
				Expect(b.Currency).ToNot(BeEmpty())
				_, perr := b.Amount.Float64()
				Expect(perr).To(BeNil(), "balance amount %q is not numeric", b.Amount.String())
			}
		})
	})

	Describe("GetBeneficiaries", func() {
		It("returns beneficiaries whose shape matches what the connector consumes", func() {
			beneficiaries, _, err := c.GetBeneficiaries(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			Expect(beneficiaries).ToNot(BeEmpty())

			for _, b := range beneficiaries {
				// ID -> PSPAccount.Reference (hard). created_at drives the
				// watermark (hard). name and currency are tolerated empty (the
				// connector only takes their addresses), so they are NOT asserted.
				Expect(b.ID).ToNot(BeEmpty())
				Expect(b.CreatedAt.IsZero()).To(BeFalse(), "beneficiary created_at is zero/unset")
			}

			if contracttest.BootstrapEnabled("CURRENCYCLOUD") {
				allIDs, err := collectAllBeneficiaryIDs(ctx, c)
				Expect(err).To(BeNil())
				contracttest.LogBootstrap("expectedBeneficiaryIDs", allIDs)
			}
		})

		It("keeps the known beneficiaries in their expected, stable relative order", func() {
			if len(expectedBeneficiaryIDs) == 0 {
				Skip("expectedBeneficiaryIDs is not populated — set CURRENCYCLOUD_CONTRACT_BOOTSTRAP=1 and fill it from the bootstrap log to enable the ordering contract")
			}

			// Beneficiaries are ordered created_at asc (immutable); new ones append
			// at the end. Assert the pinned beneficiaries retain their relative
			// order, ignoring any others, walking all pages.
			allIDs, err := collectAllBeneficiaryIDs(ctx, c)
			Expect(err).To(BeNil())
			gotKnownIDs := contracttest.FilterToPinned(allIDs, expectedBeneficiaryIDs)
			Expect(gotKnownIDs).To(Equal(expectedBeneficiaryIDs))
		})
	})

	Describe("GetTransactions", func() {
		It("returns transactions whose shape matches what the connector consumes", func() {
			// Fresh (zero) updatedAtFrom drives a full backlog read (updated_at asc).
			transactions, _, err := c.GetTransactions(ctx, 1, contractPageSize, time.Time{})
			Expect(err).To(BeNil())

			for _, tx := range transactions {
				Expect(tx.ID).ToNot(BeEmpty())
				Expect(tx.Currency).ToNot(BeEmpty())
				Expect(tx.Status).ToNot(BeEmpty())
				Expect(tx.UpdatedAt.IsZero()).To(BeFalse(), "transaction updated_at is zero/unset")
				_, perr := tx.Amount.Float64()
				Expect(perr).To(BeNil(), "transaction amount %q is not numeric", tx.Amount.String())
			}

			if contracttest.BootstrapEnabled("CURRENCYCLOUD") {
				allIDs, err := collectAllTransactionIDs(ctx, c)
				Expect(err).To(BeNil())
				contracttest.LogBootstrap("expectedTransactionIDs", allIDs)
			}
		})

		It("keeps the known transactions in their expected, stable relative order", func() {
			if len(expectedTransactionIDs) == 0 {
				Skip("expectedTransactionIDs is not populated — set CURRENCYCLOUD_CONTRACT_BOOTSTRAP=1 and fill it from the bootstrap log to enable the ordering contract")
			}

			// The money-movement specs create new transactions each run, which the
			// updated_at asc walk returns at the tail. Assert only that the *pinned*
			// (older, settled) transactions retain their relative order, ignoring any
			// newly created ones, walking all pages.
			allIDs, err := collectAllTransactionIDs(ctx, c)
			Expect(err).To(BeNil())
			gotKnownIDs := contracttest.FilterToPinned(allIDs, expectedTransactionIDs)
			Expect(gotKnownIDs).To(Equal(expectedTransactionIDs))
		})
	})

	Describe("InitiateTransfer", func() {
		// Internal transfer between two of our own demo accounts. CurrencyCloud has
		// no AllowOverdraft, so we source the currency + account from a funded
		// balance and Skip when the demo has no funded account. Smallest amount,
		// unique idempotency reference per run.
		It("initiates a minimal internal transfer", func() {
			accounts, _, err := c.GetAccounts(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			if len(accounts) < 2 {
				Skip("need at least 2 demo accounts to exercise a transfer")
			}

			balances, _, err := c.GetBalances(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			funded, ok := findFundedBalance(balances)
			if !ok {
				Skip("no funded demo account to source a transfer (CurrencyCloud has no overdraft)")
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
				SourceAccountID:      funded.accountID,
				DestinationAccountID: destination,
				Currency:             funded.currency,
				Amount:               contractMinAmount,
				Reason:               "Formance Contract Test",
				UniqueRequestID:      contracttest.Ref("currencycloud", "transfer"),
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
			Expect(resp.Currency).ToNot(BeEmpty())
			Expect(resp.CreatedAt.IsZero()).To(BeFalse(), "transfer created_at is zero/unset")
			_, perr := resp.Amount.Float64()
			Expect(perr).To(BeNil(), "transfer amount %q is not numeric", resp.Amount.String())
		})
	})

	Describe("InitiatePayout", func() {
		// Outbound payment to a beneficiary. Source account + currency derived from
		// a funded balance whose currency a beneficiary also uses; Skip when no such
		// match exists. Smallest amount, unique idempotency reference per run.
		It("initiates a minimal outbound payout", func() {
			balances, _, err := c.GetBalances(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())

			beneficiaries, _, err := c.GetBeneficiaries(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			if len(beneficiaries) == 0 {
				Skip("need at least 1 beneficiary to exercise a payout")
			}

			// The payout API (v2/payments/create) takes no source account — funds
			// are drawn from the house balance in the payment currency. Pick a
			// funded balance whose currency a beneficiary also uses, so the
			// outbound payment currency is both funded and accepted.
			var (
				currency      string
				beneficiaryID string
				matched       bool
			)
			for _, b := range balances {
				amount, aerr := b.Amount.Float64()
				if aerr != nil || amount < contractMinFunded {
					continue
				}
				for _, bene := range beneficiaries {
					if bene.Currency == b.Currency {
						currency = b.Currency
						beneficiaryID = bene.ID
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				Skip("no funded demo balance whose currency matches a beneficiary to exercise a payout")
			}

			resp, err := c.InitiatePayout(ctx, &PayoutRequest{
				BeneficiaryID:   beneficiaryID,
				Currency:        currency,
				Amount:          contractMinAmount,
				Reference:       "Formance Contract Test",
				Reason:          "Formance Contract Test",
				UniqueRequestID: contracttest.Ref("currencycloud", "payout"),
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
			Expect(resp.Currency).ToNot(BeEmpty())
			Expect(resp.BeneficiaryID).ToNot(BeEmpty())
			_, perr := resp.Amount.Float64()
			Expect(perr).To(BeNil(), "payout amount %q is not numeric", resp.Amount.String())
		})
	})
})
