//go:build contract

// Package client contract test for the Powens connector.
//
// This is a CONTRACT test: it calls the real Powens API over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types). It is gated behind the `contract` build tag so it never
// runs as part of `just tests` (which only enables `-tags it`); it runs daily
// via the contract-tests GitHub workflow.
//
// Powens is an open-banking AGGREGATION connector: accounts, balances and
// transactions arrive via webhooks after an end user links their bank through
// the Powens webview — they are not polled through the client. So there are no
// paginated list reads, no ordering pins and no bootstrap step here; the
// contract surface is the user lifecycle (create user / temporary webview
// code / delete user) and the webhook-auth lifecycle the connector performs on
// install/uninstall. Both specs are self-cleaning: every run nets zero users
// and zero webhook auth providers in the tenant.
//
// DeleteUserConnection is the one client method NOT covered: a user connection
// can only be created by a human completing the webview bank-link flow, so the
// spec cannot arrange its precondition. Its wire shape (Bearer user token,
// /2.0/users/me/... path) is shared with DeleteUser, which is covered.
//
// Run locally:
//
//	POWENS_CONTRACT_CLIENT_ID=... POWENS_CONTRACT_CLIENT_SECRET=... \
//	    POWENS_CONTRACT_CONFIGURATION_TOKEN=... \
//	    POWENS_CONTRACT_ENDPOINT=https://<domain>.biapi.pro \
//	    just contract-tests powens
//
// Without the four env vars the suite Skips rather than fails, so it is safe
// to run anywhere.
package client

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPowensContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Powens Contract Suite")
}

var _ = Describe("Powens API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		clientID := os.Getenv("POWENS_CONTRACT_CLIENT_ID")
		clientSecret := os.Getenv("POWENS_CONTRACT_CLIENT_SECRET")
		configurationToken := os.Getenv("POWENS_CONTRACT_CONFIGURATION_TOKEN")
		endpoint := os.Getenv("POWENS_CONTRACT_ENDPOINT")
		if clientID == "" || clientSecret == "" || configurationToken == "" || endpoint == "" {
			Skip("POWENS_CONTRACT_CLIENT_ID, POWENS_CONTRACT_CLIENT_SECRET, " +
				"POWENS_CONTRACT_CONFIGURATION_TOKEN and POWENS_CONTRACT_ENDPOINT " +
				"must be set to run the Powens contract test")
		}

		ctx = context.Background()
		var err error
		c, err = New("powens", clientID, clientSecret, configurationToken, endpoint)
		Expect(err).To(BeNil())
	})

	Describe("user lifecycle", func() {
		It("creates a permanent user, issues a temporary webview code, and deletes the user", func() {
			user, err := c.CreateUser(ctx)
			Expect(err).To(BeNil())

			// The delete is registered before the assertions so a failed run
			// never leaks the user; it doubles as the DeleteUser contract
			// coverage (mirrors the connector's deleteUser).
			DeferCleanup(func() {
				Expect(c.DeleteUser(ctx, DeleteUserRequest{
					AccessToken: user.AuthToken,
				})).To(BeNil())
			})

			// auth_token is stored as the user's permanent token and sent as
			// the Bearer token on every subsequent per-user call.
			Expect(user.AuthToken).ToNot(BeEmpty())
			// id_user becomes the PSPUserID (createUser strconv.Itoa's it).
			Expect(user.IdUser).To(BeNumerically(">", 0))

			// createUserLink exchanges the permanent token for a temporary
			// code embedded in the webview link handed to the end user.
			code, err := c.CreateTemporaryCode(ctx, CreateTemporaryLinkRequest{
				AccessToken: user.AuthToken,
			})
			Expect(err).To(BeNil())
			Expect(code.Code).ToNot(BeEmpty())
			// expires_in sets the temporary link token's ExpiresAt; a zero
			// value would mean an instantly-expired webview link.
			Expect(code.ExpiresIn).To(BeNumerically(">", 0))
		})
	})

	Describe("webhook auth lifecycle", func() {
		It("creates a webhook auth with an HMAC secret, finds it in the list, and deletes it", func() {
			// The connector dedups/uninstalls webhook auths by NAME (the API
			// has no filter and CreateWebhookAuth cannot attach a connector
			// ID), so a per-run-unique name keeps runs independent. It must
			// stay SHORT and ALPHANUMERIC: Powens 500s ("dataError") on
			// contracttest.Ref's longer dashed form (verified live — a
			// 13-char alnum name creates fine, the 39-char dashed one 500s).
			name := fmt.Sprintf("powenswa%d", time.Now().UnixNano())

			secretKey, err := c.CreateWebhookAuth(ctx, name)
			Expect(err).To(BeNil())

			// Registered before the secretKey assertion so the auth is
			// reclaimed even when that assert fails (the cleanup re-finds it
			// by name, so it does not depend on anything asserted below).
			var authID int
			DeferCleanup(func() {
				// Best-effort re-find by name if the spec failed before the
				// list assertion captured the ID, so a red run does not
				// accumulate auth providers in the tenant.
				if authID == 0 {
					auths, err := c.ListWebhookAuths(ctx)
					Expect(err).To(BeNil())
					for _, a := range auths {
						if a.Name == name {
							authID = a.ID
							break
						}
					}
				}
				if authID == 0 {
					return
				}
				Expect(c.DeleteWebhookAuth(ctx, authID)).To(BeNil())

				// Uninstall's contract is create→list→delete actually
				// removing the auth: re-list and assert it is gone.
				auths, err := c.ListWebhookAuths(ctx)
				Expect(err).To(BeNil())
				for _, a := range auths {
					Expect(a.Name).ToNot(Equal(name))
				}
			})

			// config.secret_key is the HMAC key verifyWebhook uses to
			// authenticate every incoming Powens webhook.
			Expect(secretKey).ToNot(BeEmpty())

			// deleteWebhooks lists ALL auth providers and matches by name to
			// find the ID to delete — so the created auth must come back in
			// the list with its name echoed and a usable id.
			auths, err := c.ListWebhookAuths(ctx)
			Expect(err).To(BeNil())
			Expect(auths).ToNot(BeEmpty())

			for _, a := range auths {
				if a.Name == name {
					Expect(a.ID).To(BeNumerically(">", 0))
					authID = a.ID
					break
				}
			}
			Expect(authID).ToNot(BeZero(),
				"created webhook auth %q not found in ListWebhookAuths", name)
		})
	})
})
