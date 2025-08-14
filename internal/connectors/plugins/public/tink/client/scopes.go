package client

type Scopes string

const (
	SCOPES_AUTHORIZATION_READ  Scopes = "authorization:read"
	SCOPES_AUTHORIZATION_GRANT Scopes = "authorization:grant"

	SCOPES_USER_CREATE Scopes = "user:create"
	SCOPES_USER_READ   Scopes = "user:read"
	SCOPES_USER_DELETE Scopes = "user:delete"

	SCOPES_CONSENTS_READONLY Scopes = "consents:readonly"

	SCOPES_PROVIDERS_READ Scopes = "providers:read"

	SCOPES_CREDENTIALS_READ    Scopes = "credentials:read"
	SCOPES_CREDENTIALS_WRITE   Scopes = "credentials:write"
	SCOPES_CREDENTIALS_REFRESH Scopes = "credentials:refresh"

	SCOPES_ACCOUNTS_READ Scopes = "accounts:read"

	SCOPES_BALANCES_READ Scopes = "balances:read"

	SCOPES_TRANSACTIONS_READ Scopes = "transactions:read"

	SCOPES_WEBHOOKS Scopes = "webhook-endpoints"
)

var allScopes = []Scopes{
	SCOPES_AUTHORIZATION_READ,
	SCOPES_AUTHORIZATION_GRANT,
	SCOPES_USER_CREATE,
	SCOPES_USER_READ,
	SCOPES_USER_DELETE,
	SCOPES_CONSENTS_READONLY,
	SCOPES_PROVIDERS_READ,
	SCOPES_CREDENTIALS_READ,
	SCOPES_CREDENTIALS_WRITE,
	SCOPES_CREDENTIALS_REFRESH,
	SCOPES_ACCOUNTS_READ,
	SCOPES_BALANCES_READ,
	SCOPES_TRANSACTIONS_READ,
	SCOPES_WEBHOOKS,
}
