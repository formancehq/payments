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

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/pkg/domain/contracttest"
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

// Ordering is asserted on the SORT KEY, not on pinned IDs. Every list read
// requests an explicit order (accounts and transactions: updated_at asc,
// beneficiaries: created_at asc) and the connector's walk consumes exactly
// that — so the contract is "the API honors its requested sort key", checked
// directly with contracttest.AssertNonDecreasing over the full walk. Pinned-ID
// lists would be maintenance (bootstrap + refills) and would flap on
// suite-created state: this suite's own money movement bumps the source and
// destination accounts' updated_at and creates new transactions, reordering
// any pin between runs without any upstream drift.

// collectAllAccountUpdatedAts pages through every account and returns the
// updated_at sort keys in list order. Bounded to avoid an accidental unbounded
// loop.
func collectAllAccountUpdatedAts(ctx context.Context, c Client) ([]int64, error) {
	var keys []int64
	for page := 1; page <= 5000; page++ {
		accounts, nextPage, err := c.GetAccounts(ctx, page, contractPageSize)
		if err != nil {
			return nil, err
		}
		if len(accounts) == 0 {
			break
		}
		for _, a := range accounts {
			keys = append(keys, a.UpdatedAt.UnixNano())
		}
		if nextPage == -1 {
			break
		}
	}
	return keys, nil
}

// collectAllBeneficiaryCreatedAts pages through every beneficiary and returns
// the created_at sort keys in list order. Bounded to avoid an accidental
// unbounded loop.
func collectAllBeneficiaryCreatedAts(ctx context.Context, c Client) ([]int64, error) {
	var keys []int64
	for page := 1; page <= 5000; page++ {
		beneficiaries, nextPage, err := c.GetBeneficiaries(ctx, page, contractPageSize)
		if err != nil {
			return nil, err
		}
		if len(beneficiaries) == 0 {
			break
		}
		for _, b := range beneficiaries {
			keys = append(keys, b.CreatedAt.UnixNano())
		}
		if nextPage == -1 {
			break
		}
	}
	return keys, nil
}

// collectAllTransactionUpdatedAts walks the transactions list the same way the
// connector does (a full backlog read: zero updatedAtFrom, updated_at asc) and
// returns the updated_at sort keys in that order. Bounded to avoid an
// accidental unbounded loop.
func collectAllTransactionUpdatedAts(ctx context.Context, c Client) ([]int64, error) {
	var keys []int64
	for page := 1; page <= 5000; page++ {
		transactions, nextPage, err := c.GetTransactions(ctx, page, contractPageSize, time.Time{})
		if err != nil {
			return nil, err
		}
		if len(transactions) == 0 {
			break
		}
		for _, tx := range transactions {
			keys = append(keys, tx.UpdatedAt.UnixNano())
		}
		if nextPage == -1 {
			break
		}
	}
	return keys, nil
}

// fundedBalance is a demo balance we can safely source a money movement from: it
// parses and holds at least contractMinFunded major units.
type fundedBalance struct {
	accountID string
	currency  string
}

// isTwoDecimalCurrency reports whether code is a 2-decimal ISO4217 currency, the
// only kind for which contractMinAmount ("0.01") is a valid minor-unit amount.
// Money-movement source/currency selection restricts to these so a funded
// 0-decimal (e.g. JPY) or 3-decimal (e.g. BHD) balance can't make CurrencyCloud
// reject "0.01" and produce a spurious failure unrelated to API drift.
func isTwoDecimalCurrency(code string) bool {
	return currency.ISO4217Currencies[code] == 2
}

// findFundedBalance returns the first balance whose amount parses and is at least
// contractMinFunded, so a contractMinAmount (0.01) movement won't hit NSF. Returns
// ok=false when no such balance exists (the caller then Skips).
func findFundedBalance(balances []*Balance) (fundedBalance, bool) {
	for _, b := range balances {
		if !isTwoDecimalCurrency(b.Currency) {
			continue
		}
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

		})

		It("returns accounts in the updated_at order the connector requests", func() {
			// The read asks for order=updated_at asc; the walk consumes that
			// order, so assert the sort key is honored across the full walk.
			keys, err := collectAllAccountUpdatedAts(ctx, c)
			Expect(err).To(BeNil())
			Expect(keys).ToNot(BeEmpty())
			contracttest.AssertNonDecreasing(keys, "accounts updated_at")
		})
	})

	Describe("GetBalances", func() {
		It("returns balances whose shape matches what the connector consumes", func() {
			balances, _, err := c.GetBalances(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			// Assert non-empty like the sibling account/beneficiary shape specs,
			// so a demo env returning zero balances fails loudly instead of
			// silently validating no schema.
			Expect(balances).ToNot(BeEmpty())

			for _, b := range balances {
				// account_id, currency and amount are all hard dependencies:
				// account_id -> AccountReference, currency -> supported-currency
				// map key, amount -> parsed via GetAmountWithPrecisionFromString.
				Expect(b.AccountID).ToNot(BeEmpty())
				Expect(b.Currency).ToNot(BeEmpty())
				contracttest.AssertDecimalAmount(b.Amount, "balance")
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

		})

		It("returns beneficiaries in the created_at order the connector requests", func() {
			// The read asks for order=created_at asc (immutable key); assert
			// the sort key is honored across the full walk.
			keys, err := collectAllBeneficiaryCreatedAts(ctx, c)
			Expect(err).To(BeNil())
			Expect(keys).ToNot(BeEmpty())
			contracttest.AssertNonDecreasing(keys, "beneficiaries created_at")
		})
	})

	Describe("GetTransactions", func() {
		It("returns transactions whose shape matches what the connector consumes", func() {
			// Fresh (zero) updatedAtFrom drives a full backlog read (updated_at asc).
			transactions, _, err := c.GetTransactions(ctx, 1, contractPageSize, time.Time{})
			Expect(err).To(BeNil())
			// Assert non-empty like the sibling shape specs (the ordering spec
			// below already relies on the backlog being non-empty), so a demo
			// env returning zero transactions fails loudly instead of silently
			// validating no schema.
			Expect(transactions).ToNot(BeEmpty())

			for _, tx := range transactions {
				Expect(tx.ID).ToNot(BeEmpty())
				Expect(tx.Currency).ToNot(BeEmpty())
				Expect(tx.Status).ToNot(BeEmpty())
				Expect(tx.UpdatedAt.IsZero()).To(BeFalse(), "transaction updated_at is zero/unset")
				contracttest.AssertDecimalAmount(tx.Amount, "transaction")
			}

		})

		It("returns transactions in the updated_at order the connector requests", func() {
			// The full-backlog walk (zero updatedAtFrom) asks for
			// order=updated_at asc — the exact order the connector's
			// incremental cursor depends on; assert the sort key is honored
			// across the full walk.
			keys, err := collectAllTransactionUpdatedAts(ctx, c)
			Expect(err).To(BeNil())
			Expect(keys).ToNot(BeEmpty())
			contracttest.AssertNonDecreasing(keys, "transactions updated_at")
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
			contracttest.AssertDecimalAmount(resp.Amount, "transfer")
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
				if !isTwoDecimalCurrency(b.Currency) {
					continue
				}
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
			contracttest.AssertDecimalAmount(resp.Amount, "payout")
		})
	})
})
