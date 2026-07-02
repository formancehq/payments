//go:build contract

// Package client contract test for the Mangopay connector.
//
// This is a CONTRACT test: it calls the real Mangopay sandbox over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types) or in list ordering. It is gated behind the `contract`
// build tag so it never runs as part of `just tests` (which only enables
// `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	MANGOPAY_CONTRACT_CLIENT_ID=... MANGOPAY_CONTRACT_API_KEY=... \
//	    just contract-tests mangopay
//
// The connector targets Mangopay's v2.01 API. Sandbox and production are
// DIFFERENT hosts (api.sandbox.mangopay.com vs api.mangopay.com) and the
// OAuth2 client-credentials pair only authenticates against one environment, so
// the endpoint is hardcoded to the sandbox host (contractEndpoint) and only the
// ClientId/API key are secrets. Auth needs BOTH, so without either env var the
// suite Skips rather than fails — safe to run anywhere.
//
// Mangopay's ingestion is scoped by User ID: there are no top-level account
// lists, so every wallet/bank-account/transaction read is discovered at runtime
// (list users, find one with wallets, use wallets[0] for the balance +
// transactions reads). Specs Skip when the sandbox has no usable user/wallet.
//
// Ordering: each consumed list is fetched with Sort=CreationDate:ASC and the
// connector derives a LastCreationDate watermark from list position, so the real
// dependency is "the API returns the list sorted by the immutable CreationDate
// ascending". We assert that directly (monotonic non-decreasing CreationDate),
// which needs no pinned-ID bootstrap and is robust to growth.
//
// Money movement: the InitiateWalletTransfer / InitiatePayout specs make REAL
// calls against the sandbox at the smallest amount (1 minor unit) with a unique
// idempotency Reference per run. Mangopay has NO overdraft, so they discover a
// funded wallet (>= 1 minor unit) by reading GetWallet balances and Skip when
// the sandbox has no funded wallet / no same-currency pair / no bank account.
// They accumulate sandbox state by design (accepted). GetRefund and the webhook
// Hook create/update paths are intentionally NOT exercised: refund IDs are not
// reliably discoverable, and Mangopay hooks are one-per-event-type with no delete
// (not safely repeatable) — their shapes are covered by the unit tests.
package client

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/contracttest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMangopayContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mangopay Contract Suite")
}

const contractPageSize = 100

// contractEndpoint is the Mangopay sandbox host. Sandbox and production are
// different hosts; the sandbox ClientId/API key only authenticate here.
const contractEndpoint = "https://api.sandbox.mangopay.com"

// contractMinFunded is the minimum wallet balance (integer minor units) a wallet
// must hold for a money-movement spec to safely source a 1-minor-unit movement
// from it (Mangopay has no overdraft).
const contractMinFunded = 1

// findUserWithWallets walks users oldest-first and returns the first user's ID
// that owns >= 1 wallet, together with that user's wallets. ok=false when no user
// in the sandbox has a wallet. Bounded to avoid an accidental unbounded loop.
func findUserWithWallets(ctx context.Context, c Client) (string, []Wallet, bool, error) {
	for page := 1; page <= 5000; page++ {
		users, err := c.GetUsers(ctx, page, contractPageSize)
		if err != nil {
			return "", nil, false, err
		}
		if len(users) == 0 {
			break
		}
		for _, u := range users {
			wallets, err := c.GetWallets(ctx, u.ID, 1, contractPageSize)
			if err != nil {
				return "", nil, false, err
			}
			if len(wallets) > 0 {
				return u.ID, wallets, true, nil
			}
		}
		if len(users) < contractPageSize {
			break
		}
	}
	return "", nil, false, nil
}

// findUserWithBankAccounts walks users oldest-first and returns the first user's
// ID that owns >= 1 bank account. ok=false when none exists in the sandbox.
func findUserWithBankAccounts(ctx context.Context, c Client) (string, bool, error) {
	for page := 1; page <= 5000; page++ {
		users, err := c.GetUsers(ctx, page, contractPageSize)
		if err != nil {
			return "", false, err
		}
		if len(users) == 0 {
			break
		}
		for _, u := range users {
			bankAccounts, err := c.GetBankAccounts(ctx, u.ID, 1, contractPageSize)
			if err != nil {
				return "", false, err
			}
			if len(bankAccounts) > 0 {
				return u.ID, true, nil
			}
		}
		if len(users) < contractPageSize {
			break
		}
	}
	return "", false, nil
}

// findWalletWithTransactions walks users→wallets and returns the first page of
// transactions from the first wallet that has any (used to source both the
// GetTransactions schema spec and the get-by-id inputs). ok=false when no wallet
// in the sandbox has a transaction.
func findWalletWithTransactions(ctx context.Context, c Client) ([]Payment, bool, error) {
	for page := 1; page <= 5000; page++ {
		users, err := c.GetUsers(ctx, page, contractPageSize)
		if err != nil {
			return nil, false, err
		}
		if len(users) == 0 {
			break
		}
		for _, u := range users {
			wallets, err := c.GetWallets(ctx, u.ID, 1, contractPageSize)
			if err != nil {
				return nil, false, err
			}
			for _, w := range wallets {
				txns, err := c.GetTransactions(ctx, w.ID, 1, contractPageSize, time.Time{})
				if err != nil {
					return nil, false, err
				}
				if len(txns) > 0 {
					return txns, true, nil
				}
			}
		}
		if len(users) < contractPageSize {
			break
		}
	}
	return nil, false, nil
}

var _ = Describe("Mangopay API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		clientID := os.Getenv("MANGOPAY_CONTRACT_CLIENT_ID")
		apiKey := os.Getenv("MANGOPAY_CONTRACT_API_KEY")
		if clientID == "" || apiKey == "" {
			Skip("MANGOPAY_CONTRACT_CLIENT_ID and MANGOPAY_CONTRACT_API_KEY must be set to run the Mangopay contract test")
		}

		ctx = context.Background()
		// Hardcoded sandbox host: sandbox and prod are different hosts and the
		// sandbox credentials only authenticate against contractEndpoint.
		c = New("mangopay", clientID, apiKey, contractEndpoint)
	})

	Describe("GetUsers", func() {
		It("returns users whose shape and order match what the connector consumes", func() {
			users, err := c.GetUsers(ctx, 1, contractPageSize)
			Expect(err).To(BeNil())
			Expect(users).ToNot(BeEmpty())

			dates := make([]int64, 0, len(users))
			for _, u := range users {
				// ID -> PSPOther.ID (hard); CreationDate drives the watermark (hard).
				Expect(u.ID).ToNot(BeEmpty())
				Expect(u.CreationDate).To(BeNumerically(">", 0), "user CreationDate is not set")
				dates = append(dates, u.CreationDate)
			}
			contracttest.AssertNonDecreasing(dates, "CreationDate order")
		})
	})

	Describe("GetWallets", func() {
		It("returns wallets whose shape and order match what the connector consumes", func() {
			_, wallets, ok, err := findUserWithWallets(ctx, c)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no user with a wallet in the sandbox")
			}

			dates := make([]int64, 0, len(wallets))
			for _, w := range wallets {
				// ID -> PSPAccount.Reference (hard); CreationDate drives the watermark
				// (hard); Currency feeds FormatAsset (hard-ish). Description is only
				// address-taken as the Name (soft) — not asserted.
				Expect(w.ID).ToNot(BeEmpty())
				Expect(w.CreationDate).To(BeNumerically(">", 0), "wallet CreationDate is not set")
				Expect(w.Currency).ToNot(BeEmpty())
				dates = append(dates, w.CreationDate)
			}
			contracttest.AssertNonDecreasing(dates, "CreationDate order")
		})
	})

	Describe("GetWallet (balance)", func() {
		It("returns a wallet balance with a numeric integer amount", func() {
			_, wallets, ok, err := findUserWithWallets(ctx, c)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no user with a wallet in the sandbox")
			}

			wallet, err := c.GetWallet(ctx, wallets[0].ID)
			Expect(err).To(BeNil())
			Expect(wallet).ToNot(BeNil())
			// Balance.Amount is parsed via big.Int.SetString(…, 10) (errors otherwise);
			// Balance.Currency feeds FormatAsset.
			Expect(wallet.Balance.Currency).ToNot(BeEmpty())
			contracttest.AssertIntegerAmount(wallet.Balance.Amount, "wallet balance")
		})
	})

	Describe("GetBankAccounts", func() {
		It("returns external bank accounts whose shape and order match what the connector consumes", func() {
			userID, ok, err := findUserWithBankAccounts(ctx, c)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no user with a bank account in the sandbox")
			}

			bankAccounts, err := c.GetBankAccounts(ctx, userID, 1, contractPageSize)
			Expect(err).To(BeNil())

			dates := make([]int64, 0, len(bankAccounts))
			for _, ba := range bankAccounts {
				// ID -> PSPAccount.Reference (hard); CreationDate drives the watermark
				// (hard). OwnerName is only address-taken as the Name (soft) — not asserted.
				Expect(ba.ID).ToNot(BeEmpty())
				Expect(ba.CreationDate).To(BeNumerically(">", 0), "bank account CreationDate is not set")
				dates = append(dates, ba.CreationDate)
			}
			contracttest.AssertNonDecreasing(dates, "CreationDate order")
		})
	})

	Describe("GetTransactions", func() {
		It("returns transactions whose shape and order match what the connector consumes", func() {
			txns, ok, err := findWalletWithTransactions(ctx, c)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no wallet with transactions in the sandbox")
			}

			dates := make([]int64, 0, len(txns))
			for _, tx := range txns {
				// Id -> PSPPayment.Reference (hard); CreationDate drives the watermark
				// (hard); DebitedFunds.Amount parsed as integer (hard); Currency feeds
				// FormatAsset. Type/Status map with an OTHER default (soft) but are always
				// present upstream — assert as cheap drift signals.
				Expect(tx.Id).ToNot(BeEmpty())
				Expect(tx.CreationDate).To(BeNumerically(">", 0), "transaction CreationDate is not set")
				Expect(tx.Type).ToNot(BeEmpty())
				Expect(tx.Status).ToNot(BeEmpty())
				Expect(tx.DebitedFunds.Currency).ToNot(BeEmpty())
				contracttest.AssertIntegerAmount(tx.DebitedFunds.Amount, "transaction")
				dates = append(dates, tx.CreationDate)
			}
			contracttest.AssertNonDecreasing(dates, "CreationDate order")
		})
	})

	Describe("ListAllHooks", func() {
		It("returns hooks whose shape matches what the connector consumes", func() {
			// The sandbox may legitimately have zero hooks; assert shape only over
			// whatever is returned (getActiveHooks reads ID/EventType/URL/Validity).
			hooks, err := c.ListAllHooks(ctx)
			Expect(err).To(BeNil())
			for _, h := range hooks {
				Expect(h.ID).ToNot(BeEmpty())
				Expect(string(h.EventType)).ToNot(BeEmpty())
				Expect(h.URL).ToNot(BeEmpty())
				Expect(h.Validity).ToNot(BeEmpty())
			}
		})
	})

	Describe("get-by-id (sourced from the wallet transactions list)", func() {
		It("fetches transfer/payout/payin by ID with the shape the webhook translators consume", func() {
			// The get-by-id methods are consumed only by TranslateWebhook. Source a
			// real resource ID from the existing transactions list (no state created):
			// a transaction's Id IS the transfer/payout/payin resource ID. Test the
			// first transaction of each type present; Skip a type that isn't present.
			// GetRefund is excluded — refund IDs are not reliably discoverable here.
			txns, ok, err := findWalletWithTransactions(ctx, c)
			Expect(err).To(BeNil())
			if !ok {
				Skip("no wallet with transactions to source get-by-id inputs")
			}

			var testedTransfer, testedPayout, testedPayin bool
			for _, tx := range txns {
				switch {
				case tx.Type == "TRANSFER" && !testedTransfer:
					transfer, err := c.GetWalletTransfer(ctx, tx.Id)
					Expect(err).To(BeNil())
					Expect(transfer.ID).ToNot(BeEmpty())
					Expect(transfer.CreationDate).To(BeNumerically(">", 0))
					Expect(transfer.Status).ToNot(BeEmpty())
					contracttest.AssertIntegerAmount(transfer.DebitedFunds.Amount, "transfer")
					testedTransfer = true
				case tx.Type == "PAYOUT" && !testedPayout:
					payout, err := c.GetPayout(ctx, tx.Id)
					Expect(err).To(BeNil())
					Expect(payout).ToNot(BeNil())
					Expect(payout.ID).ToNot(BeEmpty())
					Expect(payout.CreationDate).To(BeNumerically(">", 0))
					Expect(payout.Status).ToNot(BeEmpty())
					contracttest.AssertIntegerAmount(payout.DebitedFunds.Amount, "payout")
					testedPayout = true
				case tx.Type == "PAYIN" && !testedPayin:
					payin, err := c.GetPayin(ctx, tx.Id)
					Expect(err).To(BeNil())
					Expect(payin).ToNot(BeNil())
					Expect(payin.ID).ToNot(BeEmpty())
					Expect(payin.CreationDate).To(BeNumerically(">", 0))
					Expect(payin.Status).ToNot(BeEmpty())
					contracttest.AssertIntegerAmount(payin.DebitedFunds.Amount, "payin")
					testedPayin = true
				}
				if testedTransfer && testedPayout && testedPayin {
					break
				}
			}
			if !testedTransfer && !testedPayout && !testedPayin {
				Skip("wallet transactions had no TRANSFER/PAYOUT/PAYIN to source get-by-id inputs")
			}
		})
	})

	Describe("InitiateWalletTransfer", func() {
		// Internal transfer between two same-currency wallets owned by one user:
		// money stays on the platform. Mangopay has no overdraft, so we read each
		// wallet's balance (GetWallet) to source a funded pair. Smallest amount
		// (1 minor unit), unique idempotency Reference per run.
		It("initiates a minimal internal wallet transfer", func() {
			var (
				authorID, sourceID, destID, curr string
				found                            bool
			)
			for page := 1; page <= 5000 && !found; page++ {
				users, err := c.GetUsers(ctx, page, contractPageSize)
				Expect(err).To(BeNil())
				if len(users) == 0 {
					break
				}
				for _, u := range users {
					wallets, err := c.GetWallets(ctx, u.ID, 1, contractPageSize)
					Expect(err).To(BeNil())
					if len(wallets) < 2 {
						continue
					}

					byCurrency := map[string][]string{}
					funded := map[string]bool{}
					for _, w := range wallets {
						full, err := c.GetWallet(ctx, w.ID)
						Expect(err).To(BeNil())
						var amt big.Int
						if _, ok := amt.SetString(full.Balance.Amount.String(), 10); !ok {
							continue
						}
						byCurrency[full.Currency] = append(byCurrency[full.Currency], w.ID)
						if amt.Cmp(big.NewInt(contractMinFunded)) >= 0 {
							funded[w.ID] = true
						}
					}

					for currency, ids := range byCurrency {
						if len(ids) < 2 {
							continue
						}
						var src string
						for _, id := range ids {
							if funded[id] {
								src = id
								break
							}
						}
						if src == "" {
							continue
						}
						var dst string
						for _, id := range ids {
							if id != src {
								dst = id
								break
							}
						}
						authorID, sourceID, destID, curr, found = u.ID, src, dst, currency, true
						break
					}
					if found {
						break
					}
				}
				if len(users) < contractPageSize {
					break
				}
			}
			if !found {
				Skip("no user with two same-currency wallets and a funded source to exercise a transfer (Mangopay has no overdraft)")
			}

			// Mangopay's Idempotency-Key must be 16–36 chars, alphanumeric/dashes;
			// the connector also requires Reference to be a UUID. A per-run UUID
			// satisfies both, is unique (never collides on the idempotency key), and
			// mirrors what production sends.
			resp, err := c.InitiateWalletTransfer(ctx, &TransferRequest{
				Reference:        contracttest.UUIDRef(),
				AuthorID:         authorID,
				DebitedWalletID:  sourceID,
				CreditedWalletID: destID,
				DebitedFunds:     Funds{Currency: curr, Amount: "1"},
				Fees:             Funds{Currency: curr, Amount: "0"},
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
			Expect(resp.CreationDate).To(BeNumerically(">", 0))
			contracttest.AssertIntegerAmount(resp.DebitedFunds.Amount, "transfer")
		})
	})

	Describe("InitiatePayout (bankwire)", func() {
		// Outbound bankwire payout from a funded wallet to one of the user's bank
		// accounts. Smallest amount, unique idempotency Reference per run. NOTE:
		// BankAccount exposes no currency, so the funded wallet's currency must
		// match the bank account when the sandbox is seeded — otherwise Mangopay
		// rejects the payout. Description must be <=12 alphanumeric chars.
		It("initiates a minimal bankwire payout", func() {
			var (
				authorID, walletID, curr, bankID string
				found                            bool
			)
			for page := 1; page <= 5000 && !found; page++ {
				users, err := c.GetUsers(ctx, page, contractPageSize)
				Expect(err).To(BeNil())
				if len(users) == 0 {
					break
				}
				for _, u := range users {
					wallets, err := c.GetWallets(ctx, u.ID, 1, contractPageSize)
					Expect(err).To(BeNil())
					if len(wallets) == 0 {
						continue
					}
					bankAccounts, err := c.GetBankAccounts(ctx, u.ID, 1, contractPageSize)
					Expect(err).To(BeNil())
					if len(bankAccounts) == 0 {
						continue
					}

					for _, w := range wallets {
						full, err := c.GetWallet(ctx, w.ID)
						Expect(err).To(BeNil())
						var amt big.Int
						if _, ok := amt.SetString(full.Balance.Amount.String(), 10); !ok {
							continue
						}
						if amt.Cmp(big.NewInt(contractMinFunded)) >= 0 {
							authorID, walletID, curr, bankID, found = u.ID, w.ID, full.Currency, bankAccounts[0].ID, true
							break
						}
					}
					if found {
						break
					}
				}
				if len(users) < contractPageSize {
					break
				}
			}
			if !found {
				Skip("no user with a funded wallet and a bank account to exercise a payout (Mangopay has no overdraft)")
			}

			// Per-run UUID: satisfies Mangopay's 16–36 char Idempotency-Key rule and
			// the connector's UUID Reference requirement; unique per run.
			resp, err := c.InitiatePayout(ctx, &PayoutRequest{
				Reference:       contracttest.UUIDRef(),
				AuthorID:        authorID,
				DebitedWalletID: walletID,
				BankAccountID:   bankID,
				DebitedFunds:    Funds{Currency: curr, Amount: "1"},
				Fees:            Funds{Currency: curr, Amount: "0"},
				BankWireRef:     "Formance",
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
			Expect(resp.CreationDate).To(BeNumerically(">", 0))
			contracttest.AssertIntegerAmount(resp.DebitedFunds.Amount, "payout")
		})
	})
})
