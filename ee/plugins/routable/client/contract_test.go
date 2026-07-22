//go:build contract

// Package client contract test for the Routable connector.
//
// This is a CONTRACT test: it calls the real Routable sandbox environment over
// the network through the same client.Client the connector uses, and asserts
// that the responses the Payments project depends on have not drifted in schema
// (field presence + types) or in list ordering. It is gated behind the
// `contract` build tag so it never runs as part of `just tests` (which only
// enables `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	ROUTABLE_CONTRACT_API_KEY=... just contract-tests routable
//
// The connector targets Routable's v1 API (GA since 2022-01-12; the deprecated
// Routable API is the older pre-v1 surface). Sandbox and production are
// DIFFERENT hosts (api.sandbox.routable.com vs api.routable.com) and a sandbox
// key only authenticates against the sandbox, so the endpoint is hardcoded to
// the sandbox host and only the API key is a secret. Without the env var the
// suite Skips rather than fails, so it is safe to run anywhere.
//
// Mutation: the CreatePayable spec makes a REAL create call against the
// sandbox (which moves no money and emails no vendors). It additionally needs
// an acting_team_member UUID, which the client has no method to discover, so
// it is injected via ROUTABLE_CONTRACT_ACTING_TEAM_MEMBER and the spec Skips
// while that is unset — the mutation is opt-in per environment. It accumulates
// one sandbox payable per run by design (accepted; Routable exposes no delete).
package client

import (
	"context"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/pkg/domain/contracttest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRoutableContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Routable Contract Suite")
}

// contractSandboxEndpoint is Routable's sandbox host. Sandbox and prod are
// different hosts and a sandbox key only works against the sandbox, so this is
// hardcoded rather than injected — a leaked misconfiguration can never point
// sandbox credentials at production.
const contractSandboxEndpoint = "https://api.sandbox.routable.com"

// contractPageSize mirrors the connector's PAGE_SIZE (config.go), which
// already encodes Routable's documented page_size cap (100) across v1 list
// endpoints.
const contractPageSize = 100

// contractMinAmount is the smallest amount the create-payable spec sends:
// Routable amounts are major-unit decimal strings, so one minor unit of USD is
// "0.01" (mappers.FromMinorUnits(1, 2)).
const contractMinAmount = "0.01"

// No ordering pins anywhere: no Routable fetcher consumes the API's list
// order. The accounts/companies cursor is a bare page counter that restarts at
// 1 every cycle (pageState), and the payments cursor is a MAX-over-rows
// status_changed_at watermark (mappers.LaterOf), never derived from list
// position. A default-sort change would at worst dup/miss rows transiently
// within one walk, healed by the next full re-walk plus engine-side Reference
// dedup — so ordering is not part of the consumed contract (schema-only, the
// plaid/tink precedent).

// assertDecimalAmount asserts a Routable major-unit decimal amount string
// parses the same way mappers.ToMinorUnits does (big.Rat.SetString, which
// errors on empty/garbage input and silently DROPS the row in ingestion).
func assertDecimalAmount(amount, label string) {
	Expect(strings.TrimSpace(amount)).ToNot(BeEmpty(), "%s amount is empty", label)
	_, ok := new(big.Rat).SetString(amount)
	Expect(ok).To(BeTrue(), "%s amount %q does not parse as a decimal", label, amount)
}

// assertSupportedCurrency asserts currency_code resolves in the ISO4217 table
// mappers.PrecisionFor consults. A missing or unknown code makes PrecisionFor
// error, which silently SKIPS the payable/receivable row in ingestion —
// production data loss, exactly the drift this test exists to catch.
func assertSupportedCurrency(code, label string) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	Expect(normalized).ToNot(BeEmpty(), "%s currency_code is empty", label)
	_, ok := currency.ISO4217Currencies[normalized]
	Expect(ok).To(BeTrue(), "%s currency_code %q is not a supported ISO4217 code", label, code)
}

var _ = Describe("Routable API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		apiKey := os.Getenv("ROUTABLE_CONTRACT_API_KEY")
		if apiKey == "" {
			Skip("ROUTABLE_CONTRACT_API_KEY must be set to run the Routable contract test")
		}

		// Bound each spec so a hung or slow sandbox call fails fast instead of
		// stalling the daily CI job indefinitely.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancel)
		c = New("routable", apiKey, contractSandboxEndpoint)
	})

	Describe("ListAccounts", func() {
		It("returns settings accounts whose shape matches what the connector consumes", func() {
			resp, err := c.ListAccounts(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			Expect(resp.Results).ToNot(BeEmpty())

			for _, a := range resp.Results {
				// id -> PSPAccount.Reference (hard). created_at -> PSPAccount.CreatedAt;
				// a malformed value fails JSON unmarshal at the client layer (hard).
				// name and currency_code are tolerated empty (pointerOrNil / no
				// DefaultAsset), so they are NOT asserted.
				Expect(a.ID).ToNot(BeEmpty())
				Expect(a.CreatedAt.IsZero()).To(BeFalse(), "account %s created_at is zero/unset", a.ID)
			}
		})
	})

	Describe("GetAccount", func() {
		It("returns accounts by ID whose shape matches the balance ingestion path", func() {
			list, err := c.ListAccounts(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			if len(list.Results) == 0 {
				Skip("no settings account in the sandbox to fetch by ID")
			}

			// fetchNextBalances re-fetches EVERY account by ID; mirror that over
			// page 1. An empty available_amount is TOLERATED per account (verified
			// live: bank-type accounts return none; AccountToBalance errors and the
			// connector logs + emits no balance), so parseability is asserted only
			// when the amount is present — but at least ONE account must carry a
			// parseable amount, otherwise the whole balances surface is silently
			// empty, which is exactly what a field rename would look like.
			balanceBearing := 0
			for _, want := range list.Results {
				account, err := c.GetAccount(ctx, want.ID)
				Expect(err).To(BeNil())
				Expect(account.ID).To(Equal(want.ID))
				if strings.TrimSpace(account.TypeDetails.AvailableAmount) != "" {
					assertDecimalAmount(account.TypeDetails.AvailableAmount, "account "+account.ID+" available")
					balanceBearing++
				}
				// currency_code empty is tolerated (AccountToBalance defaults USD),
				// but a non-empty UNKNOWN code makes PrecisionFor error and drops
				// the balance.
				if account.CurrencyCode != "" {
					assertSupportedCurrency(account.CurrencyCode, "account "+account.ID)
				}
			}
			Expect(balanceBearing).To(BeNumerically(">", 0),
				"no settings account returned an available_amount — balance ingestion would emit nothing (field renamed/moved, or sandbox has no balance account)")
		})
	})

	Describe("ListCompanies", func() {
		It("returns companies whose shape matches what the connector consumes", func() {
			resp, err := c.ListCompanies(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			Expect(resp.Results).ToNot(BeEmpty())

			for _, co := range resp.Results {
				// id -> external PSPAccount.Reference (hard). created_at ->
				// PSPAccount.CreatedAt (hard). display_name/business_name are
				// tolerated empty, so they are NOT asserted.
				Expect(co.ID).ToNot(BeEmpty())
				Expect(co.CreatedAt.IsZero()).To(BeFalse(), "company %s created_at is zero/unset", co.ID)
			}
		})
	})

	Describe("ListPayables", func() {
		It("returns payables whose shape matches what the connector consumes", func() {
			// Zero statusChangedAtGte drives a full backlog read — exactly the shape
			// fetchPayablesPage issues on a fresh cycle. No ordering pin: the payments
			// cursor is a max-over-rows watermark (mappers.LaterOf), not derived from
			// list position, and the create spec mutates this list.
			resp, err := c.ListPayables(ctx, 1, contractPageSize, time.Time{})
			Expect(err).To(BeNil())

			for _, pa := range resp.Results {
				// id -> PSPPayment.Reference (hard). amount + currency_code are hard:
				// either failing ToMinorUnits/PrecisionFor silently skips the row in
				// ingestion. created_at is the watermark fallback for draft rows
				// (StatusChangedAtOrCreated). status/delivery_method are SOFT (unknown
				// values map to UNKNOWN/OTHER); status_changed_at nil is tolerated.
				Expect(pa.ID).ToNot(BeEmpty())
				assertDecimalAmount(pa.Amount, "payable "+pa.ID)
				assertSupportedCurrency(pa.CurrencyCode, "payable "+pa.ID)
				Expect(pa.CreatedAt.IsZero()).To(BeFalse(), "payable %s created_at is zero/unset", pa.ID)
			}
		})
	})

	Describe("ListReceivables", func() {
		It("returns receivables whose shape matches what the connector consumes", func() {
			resp, err := c.ListReceivables(ctx, 1, contractPageSize, time.Time{})
			Expect(err).To(BeNil())

			for _, r := range resp.Results {
				Expect(r.ID).ToNot(BeEmpty())
				assertDecimalAmount(r.Amount, "receivable "+r.ID)
				assertSupportedCurrency(r.CurrencyCode, "receivable "+r.ID)
				Expect(r.CreatedAt.IsZero()).To(BeFalse(), "receivable %s created_at is zero/unset", r.ID)
			}
		})
	})

	Describe("GetPayable", func() {
		It("returns a payable by ID whose shape matches the polling path", func() {
			list, err := c.ListPayables(ctx, 1, contractPageSize, time.Time{})
			Expect(err).To(BeNil())
			if len(list.Results) == 0 {
				Skip("no payable in the sandbox to fetch by ID")
			}
			want := list.Results[0]

			pa, err := c.GetPayable(ctx, want.ID)
			Expect(err).To(BeNil())
			Expect(pa.ID).To(Equal(want.ID))
			assertDecimalAmount(pa.Amount, "payable "+pa.ID)
			assertSupportedCurrency(pa.CurrencyCode, "payable "+pa.ID)
			Expect(pa.CreatedAt.IsZero()).To(BeFalse(), "payable %s created_at is zero/unset", pa.ID)
		})
	})

	Describe("CreatePayable", func() {
		// Opt-in mutation: POST /v1/payables requires acting_team_member (a
		// team-member UUID the client has no method to discover), so the spec is
		// gated on ROUTABLE_CONTRACT_ACTING_TEAM_MEMBER. Source account and vendor
		// are discovered at runtime; the sandbox moves no money and emails no one.
		It("creates a minimal ACH payable and reads it back", func() {
			actingTeamMember := os.Getenv("ROUTABLE_CONTRACT_ACTING_TEAM_MEMBER")
			if actingTeamMember == "" {
				Skip("ROUTABLE_CONTRACT_ACTING_TEAM_MEMBER must be set to run the create-payable contract spec")
			}

			accounts, err := c.ListAccounts(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			var withdrawFrom string
			for _, a := range accounts.Results {
				// Verified live: withdraw_from_account must be a BANK settings
				// account — Routable 400s funding_account_not_correct_type when
				// given the "balance" account. type_details.is_usable is null on
				// bank accounts (only the balance account carries it), so it is
				// deliberately NOT part of the filter. Restrict to USD (or absent,
				// ≡ USD) so contractMinAmount's 2-decimal form is valid.
				if a.IsValid && a.Type == "bank" &&
					(a.CurrencyCode == "" || strings.EqualFold(a.CurrencyCode, "USD")) {
					withdrawFrom = a.ID
					break
				}
			}
			if withdrawFrom == "" {
				Skip("no valid USD bank settings account in the sandbox to withdraw from")
			}

			companies, err := c.ListCompanies(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			var payTo string
			for _, co := range companies.Results {
				if co.IsVendor && !co.IsArchived {
					payTo = co.ID
					break
				}
			}
			if payTo == "" {
				Skip("no non-archived vendor company in the sandbox to pay")
			}

			// The plugin sets BOTH Reference and IdempotencyKey to the payment
			// initiation reference (payable_create.go), so the spec mirrors that
			// with one per-run UUID (Routable's Idempotency-Key length rules are
			// undocumented; a UUID is universally safe).
			ref := contracttest.UUIDRef()
			payable, status, err := c.CreatePayable(ctx, CreatePayableRequest{
				Type:                "ach",
				DeliveryMethod:      "ach_standard",
				PayToCompany:        payTo,
				WithdrawFromAccount: withdrawFrom,
				Amount:              contractMinAmount,
				CurrencyCode:        "USD",
				LineItems: []PayableLineItem{{
					UnitPrice:   contractMinAmount,
					Amount:      contractMinAmount,
					Quantity:    1,
					Description: "Formance contract test",
				}},
				SendOn:           nil,
				ActingTeamMember: actingTeamMember,
				Reference:        ref,
				IdempotencyKey:   ref,
			})
			Expect(err).To(BeNil())
			// The plugin BRANCHES on the status code (201 = sync full model,
			// 202 = async {id, status} + poll). A new 2xx variant is drift.
			Expect(status).To(BeElementOf(http.StatusCreated, http.StatusAccepted),
				"CreatePayable returned unexpected status %d", status)
			// A 2xx with no ID is a contract violation the plugin errors on.
			Expect(payable.ID).ToNot(BeEmpty())
			// created_at is NOT asserted on the fresh payable: verified live, the
			// 201 echo omits it (listed payables DO carry it — asserted in the
			// ListPayables spec). Ingestion tolerates this: PayableToPSPPayment
			// maps CreatedAt via StatusChangedAtOrCreated, falling back through
			// status_changed_at to the zero time without erroring. (Connector
			// improvement candidate: a just-initiated payout can ingest with an
			// epoch timestamp if status_changed_at is also absent.)
			if status == http.StatusCreated {
				assertDecimalAmount(payable.Amount, "created payable")
				assertSupportedCurrency(payable.CurrencyCode, "created payable")
			}

			// Read-back mirrors pollPayableStatus, including the documented
			// eventual-consistency window where a GET right after a 202 404s.
			var got *Payable
			Eventually(func() error {
				p, err := c.GetPayable(ctx, payable.ID)
				if err != nil {
					return err
				}
				got = p
				return nil
			}, 90*time.Second, 5*time.Second).Should(Succeed(),
				"created payable %s never became readable", payable.ID)
			Expect(got.ID).To(Equal(payable.ID))
			assertDecimalAmount(got.Amount, "read-back payable")
			assertSupportedCurrency(got.CurrencyCode, "read-back payable")
		})
	})
})
