//go:build contract

// Package client contract test for the Tink connector.
//
// This is a CONTRACT test: it calls the real Tink API over the network through
// the same client.Client the connector uses, and asserts that the responses
// the Payments project depends on have not drifted in schema (field presence +
// types). It is gated behind the `contract` build tag so it never runs as part
// of `just tests` (which only enables `-tags it`); it runs daily via the
// contract-tests GitHub workflow.
//
// Tink is an open-banking connector with two-layered auth: an app-level OAuth2
// client-credentials token (requested lazily with ALL connector scopes — the
// Tink Console app must have every scope granted or everything goes red) and a
// per-user delegated token (authorization-grant → code exchange) minted inside
// every user-scoped call, so the delegated chain is covered implicitly.
//
// Ordering is NOT a contract anywhere: accounts and transactions paginate with
// an opaque nextPageToken and the payments fetch state is a pure resumption
// cursor. So there are no pinned IDs and no bootstrap step.
//
// The ingestion-read spec needs a user with a linked bank, which only a human
// completing a Tink Link flow can produce: link the Tink Demo Bank ONCE for a
// chosen external_user_id and expose it as TINK_CONTRACT_SEEDED_USER_ID; the
// spec Skips until then.
//
// Excluded methods: DeleteUserConnection (needs a CredentialsID that only a
// completed Link flow yields, and deleting the seed's credentials would
// destroy the seed) and the four Get*Webhook payload getters (pure local JSON
// parsing of delivered webhook bodies; covered by unit tests).
//
// Run locally:
//
//	TINK_CONTRACT_CLIENT_ID=... TINK_CONTRACT_CLIENT_SECRET=... \
//	    TINK_CONTRACT_SEEDED_USER_ID=... \
//	    just contract-tests tink
//
// Without the two credential env vars the suite Skips rather than fails, so it
// is safe to run anywhere.
package client

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/pkg/domain/httpwrapper"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTinkContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tink Contract Suite")
}

const (
	// Tink has no separate sandbox host: a sandbox Console app authenticates
	// against the same production endpoint (like Column/Stripe), so only the
	// credentials are injected.
	tinkAPIEndpoint = "https://api.tink.com"

	// Market/locale pair from the plugin's supported sets.
	contractMarket = "FR"
	contractLocale = "en_US"

	// Mirrors the plugin's PAGE_SIZE (importing the plugin package here would
	// be an import cycle).
	contractPageSize = 100

	// Infinite-loop guards for the page walks; each walk stops as soon as the
	// API returns an empty nextPageToken, so these are never call counts.
	contractMaxPages = 100
)

// contractWebhookEvents mirrors the exact event-type/url-path pairs the
// connector's initWebhookConfig installs — all six enum values being accepted
// IS the install contract.
var contractWebhookEvents = []struct {
	eventType WebhookEventType
	urlPath   string
}{
	{AccountTransactionsModified, "/account-transactions-modified"},
	{AccountTransactionsDeleted, "/account-transactions-deleted"},
	{AccountBookedTransactionsModified, "/account-booked-transactions-modified"},
	{AccountCreated, "/account-created"},
	{AccountUpdated, "/account-updated"},
	{RefreshFinished, "/refresh-finished"},
}

// seededReadDecayHint explains the most likely cause when a seeded-user read
// returns a 4xx: the one-time Tink Demo Bank link has decayed (Tink consent
// expires — the Demo Bank test connection lapses within days) or
// TINK_CONTRACT_SEEDED_USER_ID does not belong to the CI Console app's user
// set. Both are environmental — NOT the upstream schema/ordering drift this
// suite exists to catch. Re-run seed-tink-contract-user.sh (colocated) to
// re-link the Demo Bank; diagnose-tink-contract-seed.sh (colocated) tells you
// which of the two it is.
const seededReadDecayHint = "seeded-user read failed with a 4xx: the Tink Demo " +
	"Bank link has likely decayed, or TINK_CONTRACT_SEEDED_USER_ID does not " +
	"match the CI Console app — run diagnose-tink-contract-seed.sh, then re-run " +
	"seed-tink-contract-user.sh to re-link (both colocated with this test)"

// skipIfSeededReadDecayed turns a 4xx from a seeded-user read into a Skip rather
// than a failure: a decayed Tink Demo Bank consent or a seed/app mismatch is an
// environmental condition, not the schema/ordering drift this suite exists to
// catch, so it must not paint a red "contract drift" build. The connector
// returns the httpwrapper 4xx sentinel unwrapped for these reads (nil error
// body), so errors.Is detects it precisely. Any non-4xx error (5xx, transport,
// decode) still fails — that could be real drift. The schema assertions after
// each read still run whenever data IS returned, so genuine drift is caught.
func skipIfSeededReadDecayed(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, httpwrapper.ErrStatusCodeClientError) {
		Skip(seededReadDecayHint)
	}
	Expect(err).To(BeNil())
}

// assertTinkAmount re-asserts the three failure modes of the plugin's
// MapTinkAmount (which cannot be imported here without an import cycle): the
// currency must be ISO4217, unscaledValue a base-10 integer and scale an int —
// each hard-errors ingestion of the balance/payment otherwise.
func assertTinkAmount(a Amount, label string) {
	_, ok := currency.ISO4217Currencies[a.CurrencyCode]
	Expect(ok).To(BeTrue(), "%s currencyCode %q is not ISO4217", label, a.CurrencyCode)
	_, ok = new(big.Int).SetString(a.Value.Value, 10)
	Expect(ok).To(BeTrue(), "%s unscaledValue %q is not a base-10 integer", label, a.Value.Value)
	_, err := strconv.Atoi(a.Value.Scale)
	Expect(err).To(BeNil(), "%s scale %q is not an integer", label, a.Value.Scale)
}

// assertAccount covers the fields fetchNextAccounts/toPSPBalance dereference:
// the id (Reference + balance AccountReference) and the booked balance amount.
func assertAccount(account Account) {
	Expect(account.ID).ToNot(BeEmpty())
	assertTinkAmount(account.Balances.Booked.Amount,
		fmt.Sprintf("account %s booked balance", account.ID))
}

var _ = Describe("Tink API contract", func() {
	var (
		ctx          context.Context
		c            Client
		seededUserID string
	)

	BeforeEach(func() {
		clientID := os.Getenv("TINK_CONTRACT_CLIENT_ID")
		clientSecret := os.Getenv("TINK_CONTRACT_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			Skip("TINK_CONTRACT_CLIENT_ID and TINK_CONTRACT_CLIENT_SECRET " +
				"must be set to run the Tink contract test")
		}
		seededUserID = os.Getenv("TINK_CONTRACT_SEEDED_USER_ID")

		// Bound each spec so a hung or slow sandbox call fails fast instead of
		// stalling the daily CI job indefinitely.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancel)
		c = New("tink", clientID, clientSecret, tinkAPIEndpoint)
	})

	Describe("user lifecycle", func() {
		It("creates a user, issues a temporary Tink Link code, and deletes the user", func() {
			// Per-run-unique external_user_id: the connector always passes a
			// fresh PSU UUID, and Tink rejects duplicates.
			externalUserID := uuid.NewString()

			user, err := c.CreateUser(ctx, externalUserID, contractMarket, contractLocale)
			Expect(err).To(BeNil())

			// Reclaims the user even on failure; doubles as the DeleteUser
			// coverage and exercises the full delegated-auth chain
			// (authorization-grant → code exchange → user/delete). Registered
			// BEFORE the shape assertions so a failing assertion cannot leak the
			// user.
			DeferCleanup(func() {
				Expect(c.DeleteUser(ctx, DeleteUserRequest{
					UserID: externalUserID,
				})).To(BeNil())
			})

			// user_id becomes the PSPUserID.
			Expect(user.UserID).ToNot(BeEmpty())

			// createUserLink embeds this code in the Tink Link URL handed to
			// the PSU; the scope set mirrors the connector's exactly.
			code, err := c.CreateTemporaryAuthorizationCode(ctx, CreateTemporaryCodeRequest{
				UserID:   externalUserID,
				Username: "Formance Contract Test",
				WantedScopes: []Scopes{
					SCOPES_AUTHORIZATION_READ,
					SCOPES_AUTHORIZATION_GRANT,
					SCOPES_CREDENTIALS_REFRESH,
					SCOPES_CREDENTIALS_READ,
					SCOPES_CREDENTIALS_WRITE,
					SCOPES_PROVIDERS_READ,
					SCOPES_USER_READ,
				},
			})
			Expect(err).To(BeNil())
			Expect(code.Code).ToNot(BeEmpty())
		})
	})

	Describe("webhook lifecycle", func() {
		It("creates and deletes the six webhook endpoints the connector installs", func() {
			// Per-run-unique base URL so runs never collide on an existing
			// endpoint. Tink is not documented to probe the callback URL at
			// create time (unlike Wise); if this ever 4xxs on example.com,
			// switch to the postman-echo self-probe pattern.
			baseURL := fmt.Sprintf("https://example.com/contract-%d", time.Now().UnixNano())

			for _, event := range contractWebhookEvents {
				resp, err := c.CreateWebhook(ctx, event.eventType, "contract-test", baseURL+event.urlPath)
				Expect(err).To(BeNil(), "creating webhook for %s", event.eventType)

				// Registered BEFORE the shape assertions so a failing assertion
				// cannot leak the created endpoint.
				webhookID := resp.ID
				DeferCleanup(func() {
					Expect(c.DeleteWebhook(ctx, webhookID)).To(BeNil())
				})

				// id is stored in the webhook config metadata and is the only
				// handle uninstall has to delete the endpoint; secret is the
				// X-Tink-Signature HMAC key verifyWebhook depends on.
				Expect(resp.ID).ToNot(BeEmpty(), "webhook %s has no id", event.eventType)
				Expect(resp.Secret).ToNot(BeEmpty(), "webhook %s has no secret", event.eventType)
			}
		})
	})

	Describe("ingestion reads", func() {
		It("lists accounts, gets one by ID, and walks its transactions", func() {
			if seededUserID == "" {
				Skip("TINK_CONTRACT_SEEDED_USER_ID not set — link the Tink Demo " +
					"Bank once via Tink Link for a chosen external_user_id and " +
					"export it to enable the ingestion-read specs")
			}

			// Full accounts page walk (the workflow's full-refresh path).
			accounts := make([]Account, 0)
			pageToken := ""
			for page := 0; ; page++ {
				Expect(page).To(BeNumerically("<", contractMaxPages),
					"accounts page walk did not terminate")
				resp, err := c.ListAccounts(ctx, seededUserID, pageToken)
				skipIfSeededReadDecayed(err)
				accounts = append(accounts, resp.Accounts...)
				if resp.NextPageToken == "" {
					break
				}
				pageToken = resp.NextPageToken
			}
			// The seeded user promises a linked Demo Bank. Zero accounts on a 200
			// (so no 4xx skip fired above) is the same environmental decay — the
			// consent lapsed and Tink stopped returning linked accounts — so skip
			// rather than paint a red drift build.
			if len(accounts) == 0 {
				Skip(seededReadDecayHint)
			}
			for _, account := range accounts {
				assertAccount(account)
			}

			// fetchNextAccounts reads exactly ONE account by ID per webhook.
			got, err := c.GetAccount(ctx, seededUserID, accounts[0].ID)
			skipIfSeededReadDecayed(err)
			Expect(got.ID).To(Equal(accounts[0].ID))
			assertAccount(got)

			// Per-account transactions page walk, exactly the connector's
			// shape: zero date bounds (unbounded) and the plugin's page size.
			// A bad dates.* format fails the decode inside the call itself,
			// so err==nil already covers the date-format contract.
			transactions := make([]Transaction, 0)
			nextPageToken := ""
			for page := 0; ; page++ {
				Expect(page).To(BeNumerically("<", contractMaxPages),
					"transactions page walk did not terminate")
				resp, err := c.ListTransactions(ctx, ListTransactionRequest{
					UserID:        seededUserID,
					AccountID:     accounts[0].ID,
					PageSize:      contractPageSize,
					NextPageToken: nextPageToken,
				})
				skipIfSeededReadDecayed(err)
				transactions = append(transactions, resp.Transactions...)
				if resp.NextPageToken == "" {
					break
				}
				nextPageToken = resp.NextPageToken
			}
			// The Demo Bank seeds transactions on its accounts.
			Expect(transactions).ToNot(BeEmpty())
			for _, tx := range transactions {
				Expect(tx.ID).ToNot(BeEmpty())
				Expect(tx.AccountID).ToNot(BeEmpty(),
					"transaction %s has no accountId (source/destination reference)", tx.ID)
				assertTinkAmount(tx.Amount, fmt.Sprintf("transaction %s", tx.ID))
			}
		})
	})
})
