# PRD: Fireblocks Connector

## Introduction

Fireblocks is an enterprise-grade digital asset custody, transfer, and settlement platform used by institutions to secure their cryptocurrency operations. Many Formance clients already use Fireblocks as their primary custody solution and need to integrate it with Formance for unified treasury management.

This feature adds Fireblocks as a new PSP connector, enabling clients to sync their vault accounts, balances, and execute transfers directly from Formance while leveraging Fireblocks' institutional-grade security infrastructure.

## Goals

- Connect Fireblocks as a payment connector in Formance
- Sync vault accounts and asset wallets from Fireblocks
- Sync balances across all vault accounts
- Enable internal transfers between vault accounts
- Enable external transfers to whitelisted addresses
- Support webhook notifications for real-time transaction updates
- Provide visibility into transaction status and history

## User Stories

### Phase 1: Core Configuration & Authentication

#### US-001: Implement Fireblocks connector configuration
**Description:** As an operator, I want to configure a Fireblocks connector so that I can connect my Fireblocks workspace to Formance.

**Acceptance Criteria:**
- [ ] Connector accepts: apiKey, privateKey (RSA PEM format), baseUrl (optional, defaults to api.fireblocks.io)
- [ ] Supports sandbox environment for testing
- [ ] Implements JWT RS256 authentication with nonce
- [ ] Credentials are securely stored and encrypted at rest
- [ ] Connector validates credentials on installation by making a test API call
- [ ] Typecheck/lint passes

#### US-002: Generate Go client from OpenAPI spec
**Description:** As a developer, I need a Go client for the Fireblocks API so that I can interact with all endpoints consistently.

**Acceptance Criteria:**
- [ ] Download official OpenAPI 3.0 spec from Fireblocks
- [ ] Generate Go client using oapi-codegen or similar tool
- [ ] Client supports all required endpoints (vaults, transactions, wallets)
- [ ] Client handles JWT RS256 authentication
- [ ] Typecheck/lint passes

### Phase 2: Vault Accounts & Balances

#### US-003: Fetch vault accounts from Fireblocks
**Description:** As a user, I want to sync my Fireblocks vault accounts so that I can see all my custody accounts in Formance.

**Acceptance Criteria:**
- [ ] Connector implements CAPABILITY_FETCH_ACCOUNTS
- [ ] Fetches all vault accounts via GET /vault/accounts
- [ ] Maps vault accounts to Formance accounts with reference = vault.id
- [ ] Preserves vault name and customer reference ID as metadata
- [ ] Handles pagination for large workspaces
- [ ] Periodic sync runs on configured polling interval
- [ ] Typecheck/lint passes

#### US-004: Fetch asset wallets within vault accounts
**Description:** As a user, I want to see all asset wallets within each vault account so that I can track individual cryptocurrency holdings.

**Acceptance Criteria:**
- [ ] Fetches asset wallets via GET /vault/accounts/{vaultAccountId}/{assetId}
- [ ] Creates sub-accounts for each asset wallet
- [ ] Preserves deposit addresses as metadata
- [ ] Maps asset IDs to standard currency codes where possible
- [ ] Typecheck/lint passes

#### US-005: Fetch balances from Fireblocks
**Description:** As a user, I want to sync my Fireblocks balances so that I can see my holdings in Formance.

**Acceptance Criteria:**
- [ ] Connector implements CAPABILITY_FETCH_BALANCES
- [ ] Fetches balances for all vault accounts and assets
- [ ] Maps total, available, pending, and frozen amounts
- [ ] Handles staking balances where applicable
- [ ] Periodic sync runs on configured polling interval
- [ ] Typecheck/lint passes

### Phase 3: External Wallets

#### US-006: Fetch external wallets from Fireblocks
**Description:** As a user, I want to see my whitelisted external wallets so that I can use them as transfer destinations.

**Acceptance Criteria:**
- [ ] Connector implements CAPABILITY_FETCH_EXTERNAL_ACCOUNTS
- [ ] Fetches external wallets via GET /external_wallets
- [ ] Maps to Formance external accounts with reference = wallet.id
- [ ] Preserves wallet name, address, and tag as metadata
- [ ] Typecheck/lint passes

#### US-007: Fetch internal wallets from Fireblocks
**Description:** As a user, I want to see internal wallets (one-time addresses) for tracking purposes.

**Acceptance Criteria:**
- [ ] Fetches internal wallets via GET /internal_wallets
- [ ] Maps to Formance accounts appropriately
- [ ] Preserves address and asset information
- [ ] Typecheck/lint passes

### Phase 4: Transactions & Transfers

#### US-008: Fetch transactions from Fireblocks
**Description:** As a user, I want to sync my Fireblocks transactions so that all transfers are visible in Formance.

**Acceptance Criteria:**
- [ ] Connector implements CAPABILITY_FETCH_PAYMENTS
- [ ] Fetches transactions via GET /transactions
- [ ] Handles pagination and filtering by status/time
- [ ] Maps transaction types: TRANSFER, INTERNAL, MINT, BURN, etc.
- [ ] Maps status: SUBMITTED, PENDING, COMPLETED, FAILED, CANCELLED, REJECTED
- [ ] Preserves fee information and network details
- [ ] Periodic sync runs on configured polling interval
- [ ] Typecheck/lint passes

#### US-009: Create internal transfer between vault accounts
**Description:** As a user, I want to transfer assets between my vault accounts within Fireblocks.

**Acceptance Criteria:**
- [ ] Connector implements CAPABILITY_CREATE_TRANSFER
- [ ] Creates transaction via POST /transactions with type=INTERNAL_TRANSFER
- [ ] Supports source and destination vault account IDs
- [ ] Supports specifying asset and amount
- [ ] Returns transaction ID for tracking
- [ ] Typecheck/lint passes

#### US-010: Create external transfer to whitelisted address
**Description:** As a user, I want to transfer assets to an external whitelisted address.

**Acceptance Criteria:**
- [ ] Creates transaction via POST /transactions with type=TRANSFER
- [ ] Validates destination is in external wallets whitelist
- [ ] Supports specifying network/chain for multi-chain assets
- [ ] Supports fee level selection (LOW, MEDIUM, HIGH)
- [ ] Returns transaction ID for tracking
- [ ] Typecheck/lint passes

#### US-011: Get transaction by ID
**Description:** As a user, I want to check the status of a specific transaction.

**Acceptance Criteria:**
- [ ] Fetches transaction via GET /transactions/{txId}
- [ ] Returns full transaction details including status, fees, hash
- [ ] Maps to Formance payment model
- [ ] Typecheck/lint passes

### Phase 5: Webhooks & Real-time Updates

#### US-012: Implement webhook handler for transaction updates
**Description:** As a system, I want to receive real-time transaction updates so that Formance reflects current state without polling delays.

**Acceptance Criteria:**
- [ ] Implements webhook endpoint for Fireblocks notifications
- [ ] Validates webhook signature using workspace public key
- [ ] Handles TRANSACTION_STATUS_UPDATED events
- [ ] Updates payment status in real-time
- [ ] Handles VAULT_ACCOUNT_ADDED and VAULT_ACCOUNT_ASSET_ADDED events
- [ ] Typecheck/lint passes

#### US-013: Configure webhook in Fireblocks workspace
**Description:** As an operator, I need documentation on setting up webhooks in Fireblocks.

**Acceptance Criteria:**
- [ ] Connector provides webhook URL format in configuration response
- [ ] Documentation explains Fireblocks webhook setup process
- [ ] Supports webhook signature verification
- [ ] Typecheck/lint passes

### Phase 6: Supported Assets & Network Info

#### US-014: Fetch supported assets from Fireblocks
**Description:** As a user, I want to see which assets are supported by my Fireblocks workspace.

**Acceptance Criteria:**
- [ ] Fetches supported assets via GET /supported_assets
- [ ] Returns asset ID, name, symbol, decimals, and supported networks
- [ ] Caches results to reduce API calls
- [ ] Typecheck/lint passes

#### US-015: Get gas station configuration
**Description:** As a user, I want to see my gas station settings for automatic fee funding.

**Acceptance Criteria:**
- [ ] Fetches gas station config via GET /gas_station
- [ ] Returns threshold and funding configuration per asset
- [ ] Preserves as connector metadata
- [ ] Typecheck/lint passes

### Phase 7: Exchange Accounts (Optional)

#### US-016: Fetch connected exchange accounts
**Description:** As a user, I want to see exchange accounts connected to my Fireblocks workspace.

**Acceptance Criteria:**
- [ ] Fetches exchange accounts via GET /exchange_accounts
- [ ] Maps to Formance accounts with type=EXCHANGE
- [ ] Preserves exchange name and trading account info
- [ ] Typecheck/lint passes

#### US-017: Fetch exchange balances
**Description:** As a user, I want to see balances on my connected exchanges.

**Acceptance Criteria:**
- [ ] Fetches balances for each exchange account
- [ ] Maps available, total, and locked amounts
- [ ] Updates during balance sync
- [ ] Typecheck/lint passes

## Functional Requirements

### Authentication
- FR-1: The connector must implement JWT RS256 authentication with API key and RSA private key
- FR-2: Each request must include a unique nonce to prevent replay attacks
- FR-3: JWT tokens must have short expiration (30 seconds recommended)

### Vault Management
- FR-4: The system must sync all vault accounts from the Fireblocks workspace
- FR-5: The system must sync all asset wallets within each vault account
- FR-6: Vault account hierarchy must be preserved in Formance account structure

### Transaction Processing
- FR-7: The system must support internal transfers between vault accounts
- FR-8: The system must support external transfers to whitelisted addresses only
- FR-9: Transaction status must follow lifecycle: SUBMITTED → PENDING_AUTHORIZATION → QUEUED → PENDING → BROADCASTING → CONFIRMING → COMPLETED (or FAILED/CANCELLED/REJECTED)
- FR-10: The system must preserve transaction hash and network fee information

### Webhooks
- FR-11: The system must validate webhook signatures using the workspace public key
- FR-12: Webhook events must be processed idempotently
- FR-13: Failed webhook processing must not block subsequent events

### API Endpoints
- FR-14: All existing payment endpoints must work with Fireblocks connector
- FR-15: Fireblocks-specific metadata must be accessible via standard metadata endpoints

## Non-Goals (Out of Scope)

- **Trading via Off-Exchange** - Only custody and transfers, not exchange trading
- **Policy management** - Fireblocks policies managed in Fireblocks console
- **User management** - Workspace users managed in Fireblocks console
- **Smart contract deployment** - Only transfers, not tokenization
- **NFT operations** - Only fungible token transfers
- **Staking operations** - Read-only staking balance visibility
- **Raw signing** - Only high-level transfer operations

## Technical Considerations

### Architecture
- Connector follows existing PSP connector pattern
- Uses generated Go client from OpenAPI spec
- Webhook handler integrated with existing webhook infrastructure

### Authentication
- JWT RS256 with RSA private key (PEM format)
- API Key identifies the API user
- Nonce (Unix timestamp in milliseconds) prevents replay attacks
- Token expiration: 30 seconds

```go
type Config struct {
    APIKey     string `json:"apiKey" bson:"apiKey"`
    PrivateKey string `json:"privateKey" bson:"privateKey"` // RSA PEM
    BaseURL    string `json:"baseUrl,omitempty" bson:"baseUrl,omitempty"`
    Sandbox    bool   `json:"sandbox,omitempty" bson:"sandbox,omitempty"`
}
```

### Rate Limiting
- Fireblocks applies per-endpoint rate limits
- Implement exponential backoff on 429 responses
- Cache supported assets to reduce API calls

### Error Handling
- Map Fireblocks error codes to standard Formance errors
- Preserve original error message in metadata for debugging
- Handle network timeouts gracefully

### Webhook Security
- Verify X-Fireblocks-Signature header
- Use workspace public key for signature verification
- Reject webhooks with invalid signatures

### Dependencies
- OpenAPI-generated client (from Fireblocks OpenAPI spec)
- OR community SDK: github.com/caxqueiroz/fireblocks-sdk (evaluate freshness)
- JWT library for RS256 signing

## Success Metrics

- Users can view all Fireblocks vault accounts and balances in Formance
- Internal transfers complete successfully with < 5 second status update latency (via webhooks)
- External transfers to whitelisted addresses complete successfully
- 99.9% webhook delivery success rate
- Zero authentication failures due to token issues

## Open Questions

1. **Client Generation vs Community SDK:** Should we generate a fresh client from OpenAPI or use/fork the community SDK?
2. **Webhook URL:** How should the webhook URL be structured for multi-tenant deployments?
3. **Policy Awareness:** Should the connector warn when a transfer might require additional approvals?
4. **Asset Mapping:** How to handle Fireblocks-specific asset IDs vs standard symbols?
5. **Sandbox Testing:** Do we have access to a Fireblocks sandbox for development?

## References

- [Fireblocks Developer Portal](https://developers.fireblocks.com/)
- [Fireblocks API Overview](https://developers.fireblocks.com/reference/api-overview)
- [Fireblocks REST API Guide](https://developers.fireblocks.com/reference/rest-api-guide-1)
- [Fireblocks Webhooks](https://developers.fireblocks.com/docs/webhooks)
- [Community Go SDK](https://github.com/caxqueiroz/fireblocks-sdk)
