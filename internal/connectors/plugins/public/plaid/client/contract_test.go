//go:build contract

// Package client contract test for the Plaid connector.
//
// This is a CONTRACT test: it calls the real Plaid sandbox over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types). It is gated behind the `contract` build tag so it never
// runs as part of `just tests` (which only enables `-tags it`); it runs daily
// via the contract-tests GitHub workflow.
//
// Plaid is an open-banking connector, but unlike Powens its ingestion DOES
// poll the API (ListAccounts, ListTransactions via /transactions/sync), and
// the sandbox exposes /sandbox/public_token/create, which mints a bank item
// for a fake institution without a human Link flow. That seeding call is
// scaffolding (made through the same authenticated SDK client the connector
// wraps) — the asserted contract is only the client.Client methods.
//
// Ordering is NOT a contract anywhere: accounts are a single unpaginated
// read with no fetch state, and the transactions fetch state is a pure
// opaque cursor (paymentsState.LastCursor = NextCursor) with no positional
// watermark. So there are no pinned IDs and no bootstrap step.
//
// Excluded methods: GetWebhookVerificationKey (its `kid` input only exists in
// the JWT header of a DELIVERED webhook; CI has no ingress),
// BaseWebhookTranslation / Translate*Webhook (pure local JSON parsing), and
// FormanceOpenBankingRedirect (targets the Formance stack, not Plaid).
//
// Run locally:
//
//	PLAID_CONTRACT_CLIENT_ID=... PLAID_CONTRACT_CLIENT_SECRET=... \
//	    just contract-tests plaid
//
// Without the two credential env vars the suite Skips rather than fails, so
// it is safe to run anywhere.
package client

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/plaid/plaid-go/v34/plaid"
)

func TestPlaidContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plaid Contract Suite")
}

const (
	// First Platypus Bank — Plaid's canonical sandbox institution; supports
	// the transactions product and always seeds accounts + transactions.
	contractSandboxInstitutionID = "ins_109508"

	// Plaid does not probe the webhook URL at link-token-create time, so a
	// syntactically valid constant is enough (delivery is not exercised).
	contractWebhookBaseURL = "https://example.com/webhooks"

	// Plaid only accepts redirect URIs allowlisted in the dashboard
	// (Dashboard → API → Allowed redirect URIs); https://example.com is
	// allowlisted in the Plaid team the contract credentials belong to. If
	// the credentials ever move to another team, allowlist it there too.
	contractRedirectURI = "https://example.com"

	contractApplicationName = "Formance Contract Test"

	// Mirrors the plugin's PAGE_SIZE (importing the plugin package here would
	// be an import cycle).
	contractPageSize = 100

	// Infinite-loop guard for the transactions cursor walk; each iteration
	// stops as soon as HasMore is false, so this is never a call count.
	contractMaxPages = 100
)

var _ = Describe("Plaid API contract", func() {
	var (
		ctx context.Context
		c   Client
		cc  *client
	)

	BeforeEach(func() {
		clientID := os.Getenv("PLAID_CONTRACT_CLIENT_ID")
		clientSecret := os.Getenv("PLAID_CONTRACT_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			Skip("PLAID_CONTRACT_CLIENT_ID and PLAID_CONTRACT_CLIENT_SECRET " +
				"must be set to run the Plaid contract test")
		}

		ctx = context.Background()
		var err error
		// isSandbox=true selects the SDK's built-in sandbox.plaid.com host.
		// The connectorID and STACK_PUBLIC_URL only feed the excluded
		// FormanceOpenBankingRedirect path, so zero values are fine.
		c, err = New("plaid", clientID, clientSecret, models.ConnectorID{}, true)
		Expect(err).To(BeNil())
		// cc gives access to the wrapped plaid.APIClient for the sandbox-only
		// seeding call; the test lives in package client to enable this.
		cc = c.(*client)
	})

	Describe("user and link-token lifecycle", func() {
		It("creates a user, issues a hosted-link token, and deletes the user", func() {
			// Per-run-unique client_user_id: Plaid rejects duplicates, and the
			// connector always passes a fresh PSU UUID.
			userID := uuid.NewString()

			userToken, err := c.CreateUser(ctx, userID)
			Expect(err).To(BeNil())
			// user_token is stored as PSU metadata and drives the link and
			// delete-user flows.
			Expect(userToken).ToNot(BeEmpty())

			// Registered before the link-token assert (which may Skip) so the
			// user is always reclaimed; it doubles as the DeleteUser coverage.
			DeferCleanup(func() {
				Expect(c.DeleteUser(ctx, userToken)).To(BeNil())
			})

			link, err := c.CreateLinkToken(ctx, CreateLinkTokenRequest{
				ApplicationName: contractApplicationName,
				UserID:          userID,
				UserToken:       userToken,
				Language:        "en",
				CountryCode:     "US",
				RedirectURI:     contractRedirectURI,
				WebhookBaseURL:  contractWebhookBaseURL,
				AttemptID:       uuid.NewString(),
			})
			Expect(err).To(BeNil())
			// LinkToken becomes the attempt's temporary token, HostedLinkUrl
			// is THE link handed to the PSU, Expiration bounds the attempt.
			Expect(link.LinkToken).ToNot(BeEmpty())
			Expect(link.HostedLinkUrl).ToNot(BeEmpty())
			Expect(link.Expiration.After(time.Now())).To(BeTrue(),
				"link token expiration %s is not in the future", link.Expiration)
		})
	})

	Describe("item lifecycle and ingestion reads", func() {
		It("exchanges a sandbox public token, lists accounts and transactions, and removes the item", func() {
			// SEEDING (not part of the contract): mint a public token for a
			// fake institution through the sandbox-only endpoint, standing in
			// for the human Link flow that normally produces it.
			seedReq := plaid.NewSandboxPublicTokenCreateRequest(
				contractSandboxInstitutionID,
				[]plaid.Products{plaid.PRODUCTS_TRANSACTIONS},
			)
			seedResp, _, err := cc.client.PlaidApi.SandboxPublicTokenCreate(ctx).
				SandboxPublicTokenCreateRequest(*seedReq).Execute()
			Expect(err).To(BeNil())
			Expect(seedResp.GetPublicToken()).ToNot(BeEmpty())

			exchanged, err := c.ExchangePublicToken(ctx, ExchangePublicTokenRequest{
				PublicToken: seedResp.GetPublicToken(),
			})
			Expect(err).To(BeNil())
			// AccessToken is the connection's credential for every later
			// read; ItemID becomes the ConnectionID.
			Expect(exchanged.AccessToken).ToNot(BeEmpty())
			Expect(exchanged.ItemID).ToNot(BeEmpty())

			// Reclaims the item even on failure; doubles as the DeleteItem
			// (connector deleteUserConnection) coverage.
			DeferCleanup(func() {
				Expect(c.DeleteItem(ctx, DeleteItemRequest{
					AccessToken: exchanged.AccessToken,
				})).To(BeNil())
			})

			accounts, err := c.ListAccounts(ctx, exchanged.AccessToken)
			Expect(err).To(BeNil())
			Expect(accounts.Accounts).ToNot(BeEmpty())

			// Production items only ever contain depository checking/savings
			// accounts (CreateLinkToken's account filter), so the ingestion
			// invariants below are asserted on that subset. The sandbox seed
			// has no filter option and links ALL of the institution's
			// accounts, including ones production can never ingest — e.g. the
			// Platypus 401k, whose 4-decimal USD balance (23631.9805) is
			// rightly rejected by TranslatePlaidAmount.
			depository := make([]plaid.AccountBase, 0, len(accounts.Accounts))
			depositoryIDs := make(map[string]struct{})
			for _, account := range accounts.Accounts {
				if account.Type != plaid.ACCOUNTTYPE_DEPOSITORY ||
					!account.Subtype.IsSet() || account.Subtype.Get() == nil {
					continue
				}
				switch *account.Subtype.Get() {
				case plaid.ACCOUNTSUBTYPE_CHECKING, plaid.ACCOUNTSUBTYPE_SAVINGS:
					depository = append(depository, account)
					depositoryIDs[account.AccountId] = struct{}{}
				}
			}
			Expect(depository).ToNot(BeEmpty(),
				"sandbox item has no checking/savings account — the production link filter would match nothing")

			for _, account := range depository {
				// Reference + the balance's AccountReference.
				Expect(account.AccountId).ToNot(BeEmpty())

				// toPSPBalance hard-errors when Current is not set, and both
				// the balance amount and the account asset die on a currency
				// TranslatePlaidAmount cannot resolve.
				Expect(account.Balances.Current.IsSet()).To(BeTrue(),
					"account %s balance.current is not set", account.AccountId)
				Expect(account.Balances.Current.Get()).ToNot(BeNil(),
					"account %s balance.current is null", account.AccountId)

				curr := account.Balances.GetIsoCurrencyCode()
				if curr == "" {
					curr = account.Balances.GetUnofficialCurrencyCode()
				}
				_, _, err := TranslatePlaidAmount(*account.Balances.Current.Get(), curr)
				Expect(err).To(BeNil(),
					"account %s currency %q is not translatable", account.AccountId, curr)
			}

			// The initial /transactions/sync pull is asynchronous, and Plaid
			// signals "not ready yet" in TWO ways (both seen live): a
			// PRODUCT_NOT_READY error, or a 200 whose next_cursor is empty
			// (with no transactions and has_more=false). Poll the first page
			// until the pull completes on both signals.
			var syncResp plaid.TransactionsSyncResponse
			Eventually(func() error {
				var err error
				syncResp, err = c.ListTransactions(ctx, exchanged.AccessToken, "", contractPageSize)
				if err != nil {
					return err
				}
				if syncResp.NextCursor == "" {
					return fmt.Errorf("initial transactions pull not finished (empty next_cursor)")
				}
				return nil
			}, "3m", "5s").Should(Succeed(),
				"transactions/sync never became ready for the sandbox item")

			// Walk the full backlog exactly like fetchNextPayments: feed
			// NextCursor back in until HasMore is false.
			added := append([]plaid.Transaction{}, syncResp.Added...)
			modified := append([]plaid.Transaction{}, syncResp.Modified...)
			removed := append([]plaid.RemovedTransaction{}, syncResp.Removed...)
			Expect(syncResp.NextCursor).ToNot(BeEmpty(),
				"next_cursor is the connector's persisted fetch state and must never be empty")
			cursor := syncResp.NextCursor

			for page := 1; syncResp.HasMore; page++ {
				Expect(page).To(BeNumerically("<", contractMaxPages),
					"transactions cursor walk did not terminate")
				syncResp, err = c.ListTransactions(ctx, exchanged.AccessToken, cursor, contractPageSize)
				Expect(err).To(BeNil())
				added = append(added, syncResp.Added...)
				modified = append(modified, syncResp.Modified...)
				removed = append(removed, syncResp.Removed...)
				Expect(syncResp.NextCursor).ToNot(BeEmpty())
				cursor = syncResp.NextCursor
			}

			// Same production-mirroring filter as the accounts: a real item
			// only carries depository checking/savings accounts, so only
			// their transactions can ever reach the connector's translation.
			depositoryTxs := make([]plaid.Transaction, 0, len(added)+len(modified))
			for _, tx := range append(added, modified...) {
				if _, ok := depositoryIDs[tx.AccountId]; ok {
					depositoryTxs = append(depositoryTxs, tx)
				}
			}

			// Platypus Bank always seeds checking-account transactions; an
			// empty backlog means the sandbox contract changed under us.
			Expect(depositoryTxs).ToNot(BeEmpty())

			for _, tx := range depositoryTxs {
				Expect(tx.TransactionId).ToNot(BeEmpty())
				Expect(tx.AccountId).ToNot(BeEmpty(),
					"transaction %s has no account_id (source/destination reference)", tx.TransactionId)

				curr := tx.GetIsoCurrencyCode()
				if curr == "" {
					curr = tx.GetUnofficialCurrencyCode()
				}
				_, _, err := TranslatePlaidAmount(math.Abs(tx.Amount), curr)
				Expect(err).To(BeNil(),
					"transaction %s currency %q is not translatable", tx.TransactionId, curr)

				// translatePlaidPaymentToPSPPayment time.Parses these two
				// date-only fields when set and errors on a bad format.
				if date, ok := tx.GetDateOk(); ok && *date != "" {
					_, err := time.Parse(time.DateOnly, *date)
					Expect(err).To(BeNil(),
						"transaction %s date %q is not YYYY-MM-DD", tx.TransactionId, *date)
				}
				if date, ok := tx.GetAuthorizedDateOk(); ok && *date != "" {
					_, err := time.Parse(time.DateOnly, *date)
					Expect(err).To(BeNil(),
						"transaction %s authorized_date %q is not YYYY-MM-DD", tx.TransactionId, *date)
				}
			}

			for _, tx := range removed {
				// fetchNextPayments turns each removed entry into a
				// PSPPaymentsToDelete keyed by TransactionId.
				Expect(tx.TransactionId).ToNot(BeEmpty())
			}

			// The update flow re-issues a hosted link against the existing
			// item's access token.
			updated, err := c.UpdateLinkToken(ctx, UpdateLinkTokenRequest{
				ApplicationName: contractApplicationName,
				AttemptID:       uuid.NewString(),
				UserID:          uuid.NewString(),
				Language:        "en",
				CountryCode:     "US",
				RedirectURI:     contractRedirectURI,
				AccessToken:     exchanged.AccessToken,
				ItemID:          exchanged.ItemID,
				WebhookBaseURL:  contractWebhookBaseURL,
			})
			Expect(err).To(BeNil())
			Expect(updated.LinkToken).ToNot(BeEmpty())
			Expect(updated.HostedLinkUrl).ToNot(BeEmpty())
			Expect(updated.Expiration.After(time.Now())).To(BeTrue(),
				"update link token expiration %s is not in the future", updated.Expiration)
		})
	})
})
