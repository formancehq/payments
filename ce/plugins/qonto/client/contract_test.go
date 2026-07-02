//go:build contract

// Package client contract test for the Qonto connector.
//
// This is a CONTRACT test: it calls the real Qonto sandbox over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types) or in list ordering. It is gated behind the `contract`
// build tag so it never runs as part of `just tests` (which only enables
// `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	QONTO_CONTRACT_CLIENT_ID=... QONTO_CONTRACT_API_KEY=... \
//	    QONTO_CONTRACT_STAGING_TOKEN=... just contract-tests qonto
//
// Auth & endpoint: Qonto authenticates with "Authorization: <login>:<secret_key>"
// (built by the client's apiTransport). Sandbox and prod are DIFFERENT hosts
// (prod thirdparty-api.qonto.com), so the sandbox host is hardcoded to
// sandboxEndpoint and never risks pointing sandbox creds at prod. The sandbox
// additionally requires the X-Qonto-Staging-Token header on EVERY request (a
// per-application credential from the Qonto Developer Portal), which the client
// already sends when constructed with a staging token — so the suite needs all
// THREE env vars and Skips unless every one is set.
//
// Dates: every timestamp the connector consumes is parsed with
// client.QontoTimeformat ("2006-01-02T15:04:05.999Z", literal Z, parsed in
// UTC) — NOT RFC3339. Amounts come in both major-unit decimal (json.Number)
// and minor-unit int64 *_cents variants; ingestion consumes ONLY the _cents
// fields (balance_cents, amount_cents). An int64 silently decodes to 0 when
// the field is dropped or renamed, so the schema assertion is the cross-field
// invariant major × 100 == cents (exact, via big.Rat) rather than a bare
// presence check.
//
// Ordering uses monotonic assertions (not pinned IDs): beneficiaries and
// transactions are requested with sort_by=updated_at:asc and both watermarks
// derive from the LAST element of the page, so "the API honors its sort key"
// is the real dependency — assert non-decreasing updated_at, nothing to
// bootstrap. Account ordering is NOT a contract: fetchNextAccounts re-sorts
// locally (sortOrgBankAccountsByUpdatedAndIdAtAsc), so only schema is asserted
// there.
//
// Scope: all 3 client.Client methods are consumed by ingestion, so all are
// covered. Qonto exposes NO mutation methods (no transfers, payouts, webhooks,
// or create-beneficiary), so this is a read-only suite.
//
// API-currency note: GET /v2/organization and GET /v2/transactions are current,
// but GET /v2/beneficiaries is listed under "Beneficiaries (deprecated)" in the
// Qonto docs (replacement: /v2/sepa/beneficiaries + /v2/international/
// beneficiaries, with a different shape). No sunset date is published yet; this
// suite is the tripwire that goes red when the sunset lands.
package client

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/contracttest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQontoContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Qonto Contract Suite")
}

// sandboxEndpoint is Qonto's third-party sandbox base URL (prod is
// thirdparty-api.qonto.com). Hardcoded so sandbox credentials never risk
// hitting production; the staging token only works against this host anyway.
const sandboxEndpoint = "https://thirdparty-sandbox.staging.qonto.co"

// contractPageSize mirrors the connector's PAGE_SIZE and client.QontoMaxPageSize
// (Qonto's documented cap). The collectAll* helpers page through the full list
// regardless, so the cap only costs round-trips.
const contractPageSize = QontoMaxPageSize

// parseQontoTime parses a timestamp exactly the way ingestion does
// (time.ParseInLocation with QontoTimeformat in UTC) and fails the spec with a
// labeled message when it does not parse — every consumed date is a HARD
// dependency because the fill/map functions error on a bad one.
func parseQontoTime(value, label string) time.Time {
	t, err := time.ParseInLocation(QontoTimeformat, value, time.UTC)
	Expect(err).To(BeNil(), "%s %q is not in Qonto time format %s", label, value, QontoTimeformat)
	return t
}

// assertCentsInvariant asserts the Qonto cross-field money contract: the
// major-unit decimal (e.g. "12.34") times 100 equals the minor-unit int64
// (1234). Ingestion consumes ONLY the _cents field, which silently decodes to
// 0 if Qonto drops or renames it — the invariant catches that without pinning
// a specific balance/amount value. Exact arithmetic via big.Rat (no float).
func assertCentsInvariant(major json.Number, cents int64, label string) {
	rat, ok := new(big.Rat).SetString(major.String())
	Expect(ok).To(BeTrue(), "%s %q is not a decimal", label, major.String())
	rat.Mul(rat, big.NewRat(100, 1))
	Expect(rat.IsInt()).To(BeTrue(), "%s %q × 100 is not an integer", label, major.String())
	Expect(rat.Num().Int64()).To(Equal(cents), "%s: major %q × 100 != cents %d", label, major.String(), cents)
}

// collectAllBeneficiaries pages through every beneficiary in list order
// (updated_at asc, the connector's sort) with a zero updatedAtFrom, the way a
// fresh fetchNextExternalAccounts walk starts. Bounded to avoid an accidental
// unbounded loop.
func collectAllBeneficiaries(ctx context.Context, c Client) ([]Beneficiary, error) {
	var all []Beneficiary
	for page := 1; page <= 5000; page++ {
		beneficiaries, err := c.GetBeneficiaries(ctx, time.Time{}, page, contractPageSize)
		if err != nil {
			return nil, err
		}
		if len(beneficiaries) == 0 {
			break
		}
		all = append(all, beneficiaries...)
		if len(beneficiaries) < contractPageSize {
			break
		}
	}
	return all, nil
}

// collectAllTransactions pages through every transaction of a bank account for
// one status (updated_at asc, zero updatedAtFrom), the way a fresh
// fetchNextPayments walk starts for that status. Bounded to avoid an accidental
// unbounded loop.
func collectAllTransactions(ctx context.Context, c Client, bankAccountID, status string) ([]Transactions, error) {
	var all []Transactions
	for page := 1; page <= 5000; page++ {
		transactions, err := c.GetTransactions(ctx, bankAccountID, time.Time{}, status, page, contractPageSize)
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

var _ = Describe("Qonto API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		clientID := os.Getenv("QONTO_CONTRACT_CLIENT_ID")
		apiKey := os.Getenv("QONTO_CONTRACT_API_KEY")
		stagingToken := os.Getenv("QONTO_CONTRACT_STAGING_TOKEN")
		if clientID == "" || apiKey == "" || stagingToken == "" {
			Skip("QONTO_CONTRACT_CLIENT_ID, QONTO_CONTRACT_API_KEY and QONTO_CONTRACT_STAGING_TOKEN must be set to run the Qonto contract test")
		}

		ctx = context.Background()
		// Hardcoded sandbox host: sandbox and prod are different hosts, and the
		// sandbox requires the staging token on every request.
		c = New("qonto", clientID, apiKey, sandboxEndpoint, stagingToken)
	})

	Describe("GetOrganization", func() {
		It("returns bank accounts whose shape matches what the connector consumes", func() {
			organization, err := c.GetOrganization(ctx)
			Expect(err).To(BeNil())
			Expect(organization).ToNot(BeNil())
			Expect(organization.BankAccounts).ToNot(BeEmpty())

			for _, account := range organization.BankAccounts {
				// id -> PSPAccount.Reference (hard). updated_at is hard:
				// fillAccounts parses it with QontoTimeformat, errors otherwise,
				// and derives the LastUpdatedAt watermark from it. currency feeds
				// FormatAsset for both the account and its balance. balance_cents
				// is the PSPBalance.Amount (int64 → asserted via the cents
				// invariant, not bare presence). name/iban/bic/status/main are
				// address-taken or metadata (tolerated), so NOT asserted.
				Expect(account.Id).ToNot(BeEmpty())
				parseQontoTime(account.UpdatedAt, "bank account updated_at")
				Expect(account.Currency).ToNot(BeEmpty())
				assertCentsInvariant(account.Balance, account.BalanceCents, "bank account balance")
			}

			// No ordering assertion: fetchNextAccounts re-sorts the list locally
			// (sortOrgBankAccountsByUpdatedAndIdAtAsc), so the API's own order is
			// not a contract the connector consumes.
		})
	})

	Describe("GetBeneficiaries", func() {
		It("returns beneficiaries whose shape matches what the connector consumes, updated_at-ascending", func() {
			beneficiaries, err := collectAllBeneficiaries(ctx, c)
			Expect(err).To(BeNil())
			if len(beneficiaries) == 0 {
				Skip("no beneficiaries in the sandbox to exercise GetBeneficiaries")
			}

			updatedAts := make([]int64, 0, len(beneficiaries))
			for _, beneficiary := range beneficiaries {
				// created_at and updated_at are hard: beneficiaryToPSPAccounts
				// parses both with QontoTimeformat and errors otherwise;
				// updated_at is also the pagination watermark. The bank_account
				// reference triple is hard in aggregate: generateAccountReference
				// needs iban(+bic), account_number+swift_sort_code, or
				// account_number+routing_number, and SILENTLY DROPS the
				// beneficiary otherwise — a rename here would silently drop every
				// row, which is exactly the drift to catch. currency feeds
				// FormatAsset. id/name are metadata/address-taken (tolerated),
				// so NOT asserted.
				parseQontoTime(beneficiary.CreatedAt, "beneficiary created_at")
				updatedAt := parseQontoTime(beneficiary.UpdatedAt, "beneficiary updated_at")
				updatedAts = append(updatedAts, updatedAt.UnixNano())

				bankAccount := beneficiary.BankAccount
				hasReference := bankAccount.Iban != "" ||
					(bankAccount.AccountNumber != "" && bankAccount.SwiftSortCode != "") ||
					(bankAccount.AccountNumber != "" && bankAccount.RoutingNumber != "")
				Expect(hasReference).To(BeTrue(),
					"beneficiary %s has none of the bank_account identity combos (iban / account_number+swift_sort_code / account_number+routing_number) and would be silently dropped by ingestion", beneficiary.Id)
				Expect(bankAccount.Currency).ToNot(BeEmpty())
			}

			// The client requests sort_by=updated_at:asc and derives its
			// watermark from the last element, so ascending updated_at is the
			// ordering contract.
			contracttest.AssertNonDecreasing(updatedAts, "beneficiary updated_at")
		})
	})

	Describe("GetTransactions", func() {
		// fetchNextPayments walks each bank account once per status (pending,
		// declined, completed) — Qonto only returns one status per query — so the
		// contract is checked per (account, status) walk. Statuses with no rows
		// are fine; the spec Skips only when the whole sandbox has none.
		It("returns transactions whose shape matches what the connector consumes, updated_at-ascending per status", func() {
			organization, err := c.GetOrganization(ctx)
			Expect(err).To(BeNil())
			Expect(organization.BankAccounts).ToNot(BeEmpty())

			statuses := []string{TransactionStatusPending, TransactionStatusDeclined, TransactionStatusCompleted}
			seen := 0
			for _, account := range organization.BankAccounts {
				for _, status := range statuses {
					transactions, err := collectAllTransactions(ctx, c, account.Id, status)
					Expect(err).To(BeNil())
					if len(transactions) == 0 {
						continue
					}
					seen += len(transactions)

					updatedAts := make([]int64, 0, len(transactions))
					for _, transaction := range transactions {
						// id -> Reference/ParentReference + the dedup key (hard).
						// updated_at and emitted_at are hard:
						// transactionsToPSPPayments parses both (errors
						// otherwise) and updated_at drives the per-status
						// watermark. currency feeds FormatAsset. status must
						// echo the requested status[] filter — the per-status
						// watermark scheme corrupts if Qonto stops honoring it.
						// bank_account_id is the SourceAccountReference and must
						// echo the query filter. amount_cents is the
						// PSPPayment.Amount (int64 → cents invariant).
						// subject_type drives the type/scheme/destination
						// switches. Nested counterparty details are only read
						// when non-nil (tolerated), so NOT asserted.
						Expect(transaction.Id).ToNot(BeEmpty())
						updatedAt := parseQontoTime(transaction.UpdatedAt, "transaction updated_at")
						updatedAts = append(updatedAts, updatedAt.UnixNano())
						parseQontoTime(transaction.EmittedAt, "transaction emitted_at")
						Expect(transaction.Currency).ToNot(BeEmpty())
						Expect(transaction.Status).To(Equal(status))
						Expect(transaction.BankAccountId).To(Equal(account.Id))
						Expect(transaction.SubjectType).ToNot(BeEmpty())
						assertCentsInvariant(transaction.Amount, transaction.AmountCents, "transaction amount")
					}

					// The client requests sort_by=updated_at:asc and derives the
					// per-status watermark from the last element, so ascending
					// updated_at is the ordering contract.
					contracttest.AssertNonDecreasing(updatedAts, "transaction updated_at ("+status+")")
				}
			}

			if seen == 0 {
				Skip("no transactions in the sandbox to exercise GetTransactions")
			}
		})
	})
})
