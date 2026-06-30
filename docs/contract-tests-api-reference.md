# Contract Tests API Reference

This document catalogs all external APIs used by the Formance Payments connectors. Each section provides the endpoints, authentication mechanisms, and sample curl statements for contract testing.

---

## Table of Contents

1. [Adyen](#1-adyen)
2. [Atlar](#2-atlar)
3. [Banking Circle](#3-banking-circle)
4. [Column](#4-column)
5. [Currency Cloud](#5-currency-cloud)
6. [Dummypay](#6-dummypay)
7. [Generic](#7-generic)
8. [Increase](#8-increase)
9. [Mangopay](#9-mangopay)
10. [Modulr](#10-modulr)
11. [Moneycorp](#11-moneycorp)
12. [Plaid](#12-plaid)
13. [Powens](#13-powens)
14. [Qonto](#14-qonto)
15. [Stripe](#15-stripe)
16. [Tink](#16-tink)
17. [Wise](#17-wise)
18. [EE (Enterprise) Connectors](#18-ee-enterprise-connectors)
19. [Configuration Quick Reference](#configuration-quick-reference)

---

## Configuration Quick Reference

This table summarizes the required and optional configuration fields for each connector.

| Connector      | Required Fields                                                                                      | Optional Fields                                            |
|----------------|------------------------------------------------------------------------------------------------------|------------------------------------------------------------|
| Adyen          | `apiKey`, `companyID`                                                                                | `liveEndpointPrefix`, `webhookUsername`, `webhookPassword` |
| Atlar          | `baseUrl`, `accessKey`, `secret`                                                                     | `pollingPeriod`                                            |
| Banking Circle | `username`, `password`, `endpoint`, `authorizationEndpoint`, `userCertificate`, `userCertificateKey` | `pollingPeriod`                                            |
| Column         | `apiKey`, `endpoint`                                                                                 | `pollingPeriod`                                            |
| Currency Cloud | `loginID`, `apiKey`, `endpoint`                                                                      | -                                                          |
| Dummypay       | `directory`                                                                                          | `linkFlowError`, `updateLinkFlowError`                     |
| Generic        | `apiKey`, `endpoint`                                                                                 | `pollingPeriod`                                            |
| Increase       | `apiKey`, `endpoint`, `webhookSharedSecret`                                                          | `pollingPeriod`                                            |
| Mangopay       | `clientID`, `apiKey`, `endpoint`                                                                     | `pollingPeriod`                                            |
| Modulr         | `apiKey`, `apiSecret`, `endpoint`                                                                    | `pollingPeriod`                                            |
| Moneycorp      | `clientID`, `apiKey`, `endpoint`                                                                     | `pollingPeriod`                                            |
| Plaid          | `clientID`, `clientSecret`                                                                           | `isSandbox`                                                |
| Powens         | `clientID`, `clientSecret`, `configurationToken`, `domain`, `maxConnectionsPerLink`, `endpoint`      | -                                                          |
| Qonto          | `clientID`, `apiKey`, `endpoint`                                                                     | `stagingToken`, `pollingPeriod`                            |
| Stripe         | `apiKey`                                                                                             | `pollingPeriod`                                            |
| Tink           | `clientID`, `clientSecret`, `endpoint`                                                               | -                                                          |
| Wise           | `apiKey`, `webhookPublicKey`                                                                         | `pollingPeriod`                                            |

---

## 1. Adyen

**Location:** `internal/connectors/plugins/public/adyen/`

**Authentication:** API Key in `X-API-Key` header

**Base URLs:**
- Test: `https://management-test.adyen.com`
- Live: `https://management-live.adyen.com` (with LiveEndpointPrefix)

### Configuration

```json
{
  "apiKey": "AQE...",                        // Required - Adyen API key
  "companyID": "YOUR_COMPANY_ID",            // Required - Adyen company identifier
  "liveEndpointPrefix": "prefix",            // Optional - For live environment (URL-encoded)
  "webhookUsername": "webhook_user",         // Optional - Basic auth username (no colon allowed)
  "webhookPassword": "webhook_pass"          // Optional - Basic auth password
}
```

### Endpoints

#### List Merchant Accounts
```bash
curl -X GET "https://management-test.adyen.com/v3/companies/{companyID}/merchantAccounts?pageSize=100" \
  -H "X-API-Key: ${ADYEN_API_KEY}" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "data": [
    {
      "id": "MerchantAccount123",
      "name": "My Merchant",
      "status": "Active"
    }
  ],
  "_links": {
    "next": { "href": "/v3/companies/{companyID}/merchantAccounts?pageSize=100&pageNumber=2" }
  }
}
```

#### Create Webhook
```bash
curl -X POST "https://management-test.adyen.com/v3/companies/{companyID}/webhooks" \
  -H "X-API-Key: ${ADYEN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "standard",
    "url": "https://your-webhook-url.com/webhooks/adyen",
    "active": true,
    "communicationFormat": "json",
    "username": "webhook_user",
    "password": "webhook_password"
  }'
```

#### Delete Webhook
```bash
curl -X DELETE "https://management-test.adyen.com/v3/companies/{companyID}/webhooks/{webhookID}" \
  -H "X-API-Key: ${ADYEN_API_KEY}"
```

### Webhook Verification

Adyen webhooks support two verification methods:
- **Basic Auth:** Username/Password in webhook config
- **HMAC Signature:** `HmacSHA256` signature in request header

---

## 2. Atlar

**Location:** `internal/connectors/plugins/public/atlar/`

**Authentication:** HTTP Basic Auth (`AccessKey:Secret`)

**Base URL:** Configurable (default: `https://api.atlar.com`)

### Configuration

```json
{
  "baseUrl": "https://api.atlar.com",        // Required - API base URL
  "accessKey": "your_access_key",            // Required - Access key for authentication
  "secret": "your_secret",                   // Required - Secret for authentication
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 500)

### Endpoints

#### List Accounts
```bash
curl -X GET "https://api.atlar.com/v1/accounts?limit=100" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

**Response:**
```json
{
  "items": [
    {
      "id": "acc_123",
      "name": "Main Account",
      "iban": "SE1234567890",
      "currency": "SEK"
    }
  ],
  "nextToken": "token_for_next_page"
}
```

#### Get Account by ID
```bash
curl -X GET "https://api.atlar.com/v1/accounts/{accountID}" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

#### List Transactions
```bash
curl -X GET "https://api.atlar.com/v1/transactions?limit=100" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

#### Get Transaction
```bash
curl -X GET "https://api.atlar.com/v1/transactions/{transactionID}" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

#### Create Counterparty (External Bank Account)
```bash
curl -X POST "https://api.atlar.com/v1/counterparties" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Beneficiary Name",
    "externalAccounts": [{
      "bank": { "bic": "SWEDSESS" },
      "identifiers": [{ "type": "IBAN", "number": "SE1234567890123456789012" }]
    }]
  }'
```

#### Get Counterparty
```bash
curl -X GET "https://api.atlar.com/v1/counterparties/{counterpartyID}" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

#### List External Accounts
```bash
curl -X GET "https://api.atlar.com/v1/external-accounts?limit=100" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

#### Get External Account
```bash
curl -X GET "https://api.atlar.com/v1/external-accounts/{externalAccountID}" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

#### Get Third Party
```bash
curl -X GET "https://api.atlar.com/v1/beta/third-parties/{thirdPartyID}" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

#### Create Credit Transfer (Payment)
```bash
curl -X POST "https://api.atlar.com/v1/credit-transfers" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "externalId": "unique-transfer-id",
    "sourceAccountId": "acc_source_123",
    "destinationExternalAccountId": "ext_acc_456",
    "amount": {
      "value": 10000,
      "currency": "SEK"
    },
    "remittanceInformation": {
      "type": "UNSTRUCTURED",
      "value": "Payment description"
    }
  }'
```

#### Get Credit Transfer by External ID
```bash
curl -X GET "https://api.atlar.com/v1/credit-transfers/get-by-external-id/{externalID}" \
  -u "${ATLAR_ACCESS_KEY}:${ATLAR_SECRET}"
```

---

## 3. Banking Circle

**Location:** `internal/connectors/plugins/public/bankingcircle/`

**Authentication:** mTLS (X.509 Client Certificates) + OAuth2 Bearer Token

**Base URLs:**
- API: Configurable endpoint
- Authorization: Separate auth endpoint

### Configuration

```json
{
  "username": "your_username",               // Required - API username
  "password": "your_password",               // Required - API password
  "endpoint": "https://api.bankingcircle.com", // Required - API endpoint
  "authorizationEndpoint": "https://auth.bankingcircle.com", // Required - OAuth endpoint
  "userCertificate": "-----BEGIN CERTIFICATE-----...", // Required - X.509 client cert (PEM)
  "userCertificateKey": "-----BEGIN RSA PRIVATE KEY-----...", // Required - Private key (PEM)
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 5000)

### OAuth2 Token Request
```bash
curl -X POST "${BANKING_CIRCLE_AUTH_ENDPOINT}/oauth2/token" \
  --cert "${CLIENT_CERT_PATH}" \
  --key "${CLIENT_KEY_PATH}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&username=${USERNAME}&password=${PASSWORD}"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### Endpoints

#### List Accounts
```bash
curl -X GET "${BANKING_CIRCLE_ENDPOINT}/api/v1/accounts?PageSize=100" \
  --cert "${CLIENT_CERT_PATH}" \
  --key "${CLIENT_KEY_PATH}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

**Response:**
```json
{
  "result": [
    {
      "accountId": "12345",
      "accountDescription": "Main Account",
      "currency": "EUR",
      "status": "Open"
    }
  ],
  "pageInfo": {
    "currentPage": 1,
    "pageSize": 100
  }
}
```

#### Get Account
```bash
curl -X GET "${BANKING_CIRCLE_ENDPOINT}/api/v1/accounts/{accountID}" \
  --cert "${CLIENT_CERT_PATH}" \
  --key "${CLIENT_KEY_PATH}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Get Payment Status
```bash
curl -X GET "${BANKING_CIRCLE_ENDPOINT}/api/v1/payments/singles/{paymentID}" \
  --cert "${CLIENT_CERT_PATH}" \
  --key "${CLIENT_KEY_PATH}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Get Payment Status Details
```bash
curl -X GET "${BANKING_CIRCLE_ENDPOINT}/api/v1/payments/singles/{paymentID}/status" \
  --cert "${CLIENT_CERT_PATH}" \
  --key "${CLIENT_KEY_PATH}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Create Transfer
```bash
curl -X POST "${BANKING_CIRCLE_ENDPOINT}/api/v1/accounts/{accountID}/transfers" \
  --cert "${CLIENT_CERT_PATH}" \
  --key "${CLIENT_KEY_PATH}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "idempotencyKey": "unique-key-123",
    "amount": {
      "currency": "EUR",
      "amount": 100.00
    },
    "creditorAccount": {
      "account": "DE89370400440532013000"
    },
    "ultimateDebtor": {
      "name": "Debtor Name"
    }
  }'
```

#### Create Bank Account (Beneficiary)
```bash
curl -X POST "${BANKING_CIRCLE_ENDPOINT}/api/v1/accounts/{accountID}/beneficiaries" \
  --cert "${CLIENT_CERT_PATH}" \
  --key "${CLIENT_KEY_PATH}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Beneficiary Name",
    "iban": "DE89370400440532013000",
    "bic": "COBADEFFXXX"
  }'
```

---

## 4. Column

**Location:** `internal/connectors/plugins/public/column/`

**Authentication:** HTTP Basic Auth (`:APIKey` - empty username, API key as password)

**Base URL:** Configurable (e.g., `https://api.column.com`)

### Configuration

```json
{
  "apiKey": "your_api_key",                  // Required - Column API key
  "endpoint": "https://api.column.com",      // Required - API endpoint (must be valid URL)
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100

### Endpoints

#### List Accounts
```bash
curl -X GET "${COLUMN_ENDPOINT}/accounts?limit=100" \
  -u ":${COLUMN_API_KEY}"
```

**Response:**
```json
{
  "accounts": [
    {
      "id": "acct_123",
      "description": "Main Account",
      "currency_code": "USD",
      "type": "checking"
    }
  ],
  "has_more": true,
  "cursor": "next_page_cursor"
}
```

#### Get Account Balance
```bash
curl -X GET "${COLUMN_ENDPOINT}/accounts/{accountID}/balances" \
  -u ":${COLUMN_API_KEY}"
```

#### List Counterparties
```bash
curl -X GET "${COLUMN_ENDPOINT}/counterparties?limit=100" \
  -u ":${COLUMN_API_KEY}"
```

#### List Transactions
```bash
curl -X GET "${COLUMN_ENDPOINT}/transactions?limit=100&timeline=posted" \
  -u ":${COLUMN_API_KEY}"
```

Note: Timeline can be `posted` or `pending`

#### Create Transfer
```bash
curl -X POST "${COLUMN_ENDPOINT}/transfers" \
  -u ":${COLUMN_API_KEY}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "amount": 10000,
    "currency_code": "USD",
    "sender_account_id": "acct_sender_123",
    "receiver_account_id": "acct_receiver_456",
    "description": "Transfer description"
  }'
```

#### Create Payout
```bash
curl -X POST "${COLUMN_ENDPOINT}/payouts" \
  -u ":${COLUMN_API_KEY}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "amount": 10000,
    "currency_code": "USD",
    "account_id": "acct_123",
    "counterparty_id": "cp_456",
    "description": "Payout description"
  }'
```

#### Reverse Payout
```bash
curl -X POST "${COLUMN_ENDPOINT}/payouts/{payoutID}/reverse" \
  -u ":${COLUMN_API_KEY}" \
  -H "Idempotency-Key: unique-key-456"
```

#### Create Counterparty (Bank Account)
```bash
curl -X POST "${COLUMN_ENDPOINT}/counterparties" \
  -u ":${COLUMN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "routing_number": "021000021",
    "account_number": "123456789",
    "account_type": "checking",
    "name": "Beneficiary Name"
  }'
```

#### Create Webhook Subscription
```bash
curl -X POST "${COLUMN_ENDPOINT}/event-subscriptions" \
  -u ":${COLUMN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-webhook-url.com/webhooks/column",
    "enabled_events": ["*"]
  }'
```

#### List Webhooks
```bash
curl -X GET "${COLUMN_ENDPOINT}/event-subscriptions" \
  -u ":${COLUMN_API_KEY}"
```

#### Delete Webhook
```bash
curl -X DELETE "${COLUMN_ENDPOINT}/event-subscriptions/{eventID}" \
  -u ":${COLUMN_API_KEY}"
```

---

## 5. Currency Cloud

**Location:** `internal/connectors/plugins/public/currencycloud/`

**Authentication:** OAuth2 (API Key) -> X-Auth-Token header

**Base URL:** Configurable (default: `https://devapi.currencycloud.com`)

### Configuration

```json
{
  "loginID": "your_login_id",                // Required - CurrencyCloud login ID
  "apiKey": "your_api_key",                  // Required - CurrencyCloud API key
  "endpoint": "https://devapi.currencycloud.com" // Required - API endpoint
}
```

**Page Size:** 25

### OAuth2 Token Request
```bash
curl -X POST "https://devapi.currencycloud.com/v2/authenticate/api" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "login_id=${LOGIN_ID}&api_key=${API_KEY}"
```

**Response:**
```json
{
  "auth_token": "abc123def456..."
}
```

### Endpoints

#### List Accounts
```bash
curl -X GET "https://devapi.currencycloud.com/v2/accounts/find?per_page=100&page=1" \
  -H "X-Auth-Token: ${AUTH_TOKEN}"
```

**Response:**
```json
{
  "accounts": [
    {
      "id": "5a7b9c3d-...",
      "account_name": "Main Account",
      "status": "enabled"
    }
  ],
  "pagination": {
    "total_entries": 50,
    "total_pages": 1,
    "current_page": 1,
    "per_page": 100
  }
}
```

#### Get Balances
```bash
curl -X GET "https://devapi.currencycloud.com/v2/accounts/balances/find?per_page=100&page=1" \
  -H "X-Auth-Token: ${AUTH_TOKEN}"
```

#### List Beneficiaries
```bash
curl -X GET "https://devapi.currencycloud.com/v2/beneficiaries/find?per_page=100&page=1" \
  -H "X-Auth-Token: ${AUTH_TOKEN}"
```

#### Create Beneficiary
```bash
curl -X POST "https://devapi.currencycloud.com/v2/beneficiaries/create" \
  -H "X-Auth-Token: ${AUTH_TOKEN}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "bank_account_holder_name=John Doe&bank_country=GB&currency=GBP&name=John Doe&beneficiary_country=GB&account_number=12345678&routing_code_type_1=sort_code&routing_code_value_1=123456"
```

#### List Transactions
```bash
curl -X GET "https://devapi.currencycloud.com/v2/transactions/find?per_page=100&page=1&updated_at_from=${UPDATED_AT_FROM}" \
  -H "X-Auth-Token: ${AUTH_TOKEN}"
```

#### Create Transfer
```bash
curl -X POST "https://devapi.currencycloud.com/v2/transfers/create" \
  -H "X-Auth-Token: ${AUTH_TOKEN}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "source_account_id=${SOURCE_ACCOUNT}&destination_account_id=${DEST_ACCOUNT}&currency=GBP&amount=100.00&unique_request_id=${IDEMPOTENCY_KEY}"
```

#### Create Payment (Payout)
```bash
curl -X POST "https://devapi.currencycloud.com/v2/payments/create" \
  -H "X-Auth-Token: ${AUTH_TOKEN}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "currency=GBP&beneficiary_id=${BENEFICIARY_ID}&amount=100.00&reason=Payment&reference=REF123&unique_request_id=${IDEMPOTENCY_KEY}"
```

---

## 6. Dummypay

**Location:** `internal/connectors/plugins/public/dummypay/`

**Type:** Test/Mock connector for development and testing purposes.

No external API calls - implements in-memory mock operations.

### Configuration

```json
{
  "directory": "/path/to/data",              // Required - Directory for mock data storage
  "linkFlowError": false,                    // Optional - Simulate link flow errors
  "updateLinkFlowError": false               // Optional - Simulate update link flow errors
}
```

---

## 7. Generic

**Location:** `internal/connectors/plugins/public/generic/`

**Type:** OpenAPI-based generic connector

**Authentication:** Depends on implementation

**Base URL:** Configurable via `Endpoint` config

Uses generated OpenAPI client for flexible custom PSP integrations.

### Configuration

```json
{
  "apiKey": "your_api_key",                  // Required - API key for authentication
  "endpoint": "https://your-psp.com/api",    // Required - API endpoint
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100

---

## 8. Increase

**Location:** `internal/connectors/plugins/public/increase/`

**Authentication:** Bearer Token in Authorization header

**Base URL:** Configurable (e.g., `https://api.increase.com`)

### Configuration

```json
{
  "apiKey": "your_api_key",                  // Required - Increase API key
  "endpoint": "https://api.increase.com",    // Required - API endpoint
  "webhookSharedSecret": "whsec_...",        // Required - Webhook signature secret
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 100)

### Endpoints

#### List Accounts
```bash
curl -X GET "${INCREASE_ENDPOINT}/accounts?limit=100" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

**Response:**
```json
{
  "data": [
    {
      "id": "account_123",
      "name": "Main Account",
      "status": "open",
      "currency": "USD"
    }
  ],
  "next_cursor": "cursor_for_next_page"
}
```

#### Get Account Balance
```bash
curl -X GET "${INCREASE_ENDPOINT}/accounts/{accountID}/balance" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### List External Accounts
```bash
curl -X GET "${INCREASE_ENDPOINT}/external-accounts?limit=100" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### List Transactions
```bash
curl -X GET "${INCREASE_ENDPOINT}/transactions?limit=100&created_at[gte]=${CREATED_AT_FROM}" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### Get Transaction
```bash
curl -X GET "${INCREASE_ENDPOINT}/transactions/{transactionID}" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### List Pending Transactions
```bash
curl -X GET "${INCREASE_ENDPOINT}/pending-transactions?limit=100" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### Get Pending Transaction
```bash
curl -X GET "${INCREASE_ENDPOINT}/pending-transactions/{transactionID}" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### List Declined Transactions
```bash
curl -X GET "${INCREASE_ENDPOINT}/declined-transactions?limit=100" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### Get Declined Transaction
```bash
curl -X GET "${INCREASE_ENDPOINT}/declined-transactions/{transactionID}" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### Create Transfer
```bash
curl -X POST "${INCREASE_ENDPOINT}/transfers" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "account_id": "account_sender_123",
    "destination_account_id": "account_receiver_456",
    "amount": 10000,
    "description": "Transfer description"
  }'
```

#### Create ACH Payout
```bash
curl -X POST "${INCREASE_ENDPOINT}/ach-transfers" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "account_id": "account_123",
    "external_account_id": "external_account_456",
    "amount": 10000,
    "statement_descriptor": "Payout description"
  }'
```

#### Create Wire Payout
```bash
curl -X POST "${INCREASE_ENDPOINT}/wire-transfers" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "account_id": "account_123",
    "external_account_id": "external_account_456",
    "amount": 10000,
    "message_to_recipient": "Wire payout description"
  }'
```

#### Create Bank Account
```bash
curl -X POST "${INCREASE_ENDPOINT}/external-accounts" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "routing_number": "021000021",
    "account_number": "123456789",
    "funding": "checking",
    "description": "External Account"
  }'
```

#### Create Webhook
```bash
curl -X POST "${INCREASE_ENDPOINT}/event-subscriptions" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-webhook-url.com/webhooks/increase",
    "shared_secret": "${WEBHOOK_SHARED_SECRET}"
  }'
```

#### List Webhooks
```bash
curl -X GET "${INCREASE_ENDPOINT}/event-subscriptions" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}"
```

#### Update Webhook
```bash
curl -X PATCH "${INCREASE_ENDPOINT}/event-subscriptions/{webhookID}" \
  -H "Authorization: Bearer ${INCREASE_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "active"
  }'
```

---

## 9. Mangopay

**Location:** `internal/connectors/plugins/public/mangopay/`

**Authentication:** OAuth2 Client Credentials Flow

**Base URL:** Configurable (e.g., `https://api.sandbox.mangopay.com`)

### Configuration

```json
{
  "clientID": "your_client_id",              // Required - Mangopay client ID
  "apiKey": "your_api_key",                  // Required - Mangopay API key
  "endpoint": "https://api.sandbox.mangopay.com", // Required - API endpoint
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 100)

### OAuth2 Token Request
```bash
curl -X POST "${MANGOPAY_ENDPOINT}/v2.01/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "${CLIENT_ID}:${API_KEY}" \
  -d "grant_type=client_credentials"
```

**Response:**
```json
{
  "access_token": "abc123...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### Endpoints

#### List Users
```bash
curl -X GET "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/users?per_page=100&page=1" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

**Response:**
```json
[
  {
    "Id": "user_123",
    "PersonType": "NATURAL",
    "Email": "user@example.com"
  }
]
```

#### List Wallets for User
```bash
curl -X GET "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/users/{userID}/wallets?per_page=100&page=1" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Get Wallet
```bash
curl -X GET "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/wallets/{walletID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### List Wallet Transactions
```bash
curl -X GET "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/wallets/{walletID}/transactions?per_page=100&page=1&Sort=CreationDate:ASC&AfterDate=${AFTER_DATE}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Create Payout (Bank Wire)
```bash
curl -X POST "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/payouts/bankwire" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "AuthorId": "user_123",
    "DebitedFunds": {
      "Currency": "EUR",
      "Amount": 10000
    },
    "Fees": {
      "Currency": "EUR",
      "Amount": 0
    },
    "DebitedWalletId": "wallet_456",
    "BankAccountId": "bank_789",
    "BankWireRef": "Reference"
  }'
```

#### Get Payout Status
```bash
curl -X GET "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/payouts/{payoutID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Get Payin (Payment)
```bash
curl -X GET "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/payins/{payinID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Create IBAN Bank Account
```bash
curl -X POST "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/users/{userID}/bankaccounts/iban" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "OwnerName": "John Doe",
    "OwnerAddress": {
      "AddressLine1": "123 Main St",
      "City": "Paris",
      "PostalCode": "75001",
      "Country": "FR"
    },
    "IBAN": "FR7630006000011234567890189",
    "BIC": "BNPAFRPP"
  }'
```

#### Create US Bank Account
```bash
curl -X POST "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/users/{userID}/bankaccounts/us" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "OwnerName": "John Doe",
    "OwnerAddress": {
      "AddressLine1": "123 Main St",
      "City": "New York",
      "PostalCode": "10001",
      "Region": "NY",
      "Country": "US"
    },
    "AccountNumber": "123456789",
    "ABA": "021000021"
  }'
```

#### Create CA Bank Account
```bash
curl -X POST "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/users/{userID}/bankaccounts/ca" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "OwnerName": "John Doe",
    "OwnerAddress": {...},
    "AccountNumber": "123456789",
    "BankName": "Royal Bank",
    "InstitutionNumber": "003",
    "BranchCode": "12345"
  }'
```

#### Create GB Bank Account
```bash
curl -X POST "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/users/{userID}/bankaccounts/gb" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "OwnerName": "John Doe",
    "OwnerAddress": {...},
    "AccountNumber": "12345678",
    "SortCode": "123456"
  }'
```

#### Create Other Bank Account
```bash
curl -X POST "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/users/{userID}/bankaccounts/other" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "OwnerName": "John Doe",
    "OwnerAddress": {...},
    "AccountNumber": "123456789",
    "BIC": "COBADEFFXXX",
    "Country": "DE"
  }'
```

#### List Bank Accounts
```bash
curl -X GET "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/users/{userID}/bankaccounts?per_page=100&page=1" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Create Webhook
```bash
curl -X POST "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/hooks" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "EventType": "PAYIN_NORMAL_SUCCEEDED",
    "Url": "https://your-webhook-url.com/webhooks/mangopay"
  }'
```

#### Update Webhook
```bash
curl -X PUT "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/hooks/{hookID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "Status": "ENABLED"
  }'
```

#### List Webhooks
```bash
curl -X GET "${MANGOPAY_ENDPOINT}/v2.01/${CLIENT_ID}/hooks?per_page=100&page=1" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

---

## 10. Modulr

**Location:** `internal/connectors/plugins/public/modulr/`

**Authentication:** HTTP Basic Auth (clientID + APIKey)

**Base URL:** Configurable

### Configuration

```json
{
  "apiKey": "your_api_key",                  // Required - Modulr API key
  "apiSecret": "your_api_secret",            // Required - Modulr API secret
  "endpoint": "https://api.modulrfinance.com", // Required - API endpoint
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 500)

### Endpoints

#### List Accounts
```bash
curl -X GET "${MODULR_ENDPOINT}/accounts?size=100" \
  -u "${CLIENT_ID}:${API_KEY}"
```

**Response:**
```json
{
  "content": [
    {
      "id": "A123456",
      "name": "Main Account",
      "currency": "GBP",
      "status": "ACTIVE"
    }
  ],
  "page": 0,
  "size": 100,
  "totalSize": 50
}
```

#### Get Account
```bash
curl -X GET "${MODULR_ENDPOINT}/accounts/{accountID}" \
  -u "${CLIENT_ID}:${API_KEY}"
```

#### List Account Transactions
```bash
curl -X GET "${MODULR_ENDPOINT}/accounts/{accountID}/transactions?size=100&fromCreatedDate=${FROM_DATE}" \
  -u "${CLIENT_ID}:${API_KEY}"
```

#### List Beneficiaries
```bash
curl -X GET "${MODULR_ENDPOINT}/beneficiaries?size=100" \
  -u "${CLIENT_ID}:${API_KEY}"
```

#### List Payments
```bash
curl -X GET "${MODULR_ENDPOINT}/payments?size=100&fromCreatedDate=${FROM_DATE}" \
  -u "${CLIENT_ID}:${API_KEY}"
```

#### Create Payment (Transfer/Payout)
```bash
curl -X POST "${MODULR_ENDPOINT}/payments" \
  -u "${CLIENT_ID}:${API_KEY}" \
  -H "Content-Type: application/json" \
  -H "x-mod-nonce: unique-key-123" \
  -d '{
    "sourceAccountId": "A123456",
    "destination": {
      "type": "BENEFICIARY",
      "id": "B789012"
    },
    "currency": "GBP",
    "amount": 100.00,
    "reference": "Payment reference"
  }'
```

#### Get Payment Status
```bash
curl -X GET "${MODULR_ENDPOINT}/payments?id={paymentID}" \
  -u "${CLIENT_ID}:${API_KEY}"
```

---

## 11. Moneycorp

**Location:** `internal/connectors/plugins/public/moneycorp/`

**Authentication:** OAuth2 Client Credentials Flow

**Base URL:** Configurable

### Configuration

```json
{
  "clientID": "your_client_id",              // Required - Moneycorp client ID
  "apiKey": "your_api_key",                  // Required - Moneycorp API key (client secret)
  "endpoint": "https://api.moneycorp.com",   // Required - API endpoint
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 10000)

### OAuth2 Token Request
```bash
curl -X POST "${MONEYCORP_ENDPOINT}/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=${CLIENT_ID}&client_secret=${API_KEY}"
```

### Endpoints

#### List Accounts
```bash
curl -X GET "${MONEYCORP_ENDPOINT}/accounts?page[size]=100" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

**Response:**
```json
{
  "data": [
    {
      "id": "account_123",
      "attributes": {
        "accountName": "Main Account",
        "currency": "GBP"
      }
    }
  ],
  "meta": {
    "pagination": {
      "currentPage": 1,
      "totalPages": 1
    }
  }
}
```

#### Get Account Balances
```bash
curl -X GET "${MONEYCORP_ENDPOINT}/accounts/{accountID}/balances" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Search Transactions
```bash
curl -X GET "${MONEYCORP_ENDPOINT}/accounts/{accountID}/transactions/find?page[size]=100&filter[fromDateTime]=${FROM_DATE}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### List Recipients
```bash
curl -X GET "${MONEYCORP_ENDPOINT}/accounts/{accountID}/recipients?page[size]=100" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Create Recipient
```bash
curl -X POST "${MONEYCORP_ENDPOINT}/accounts/{accountID}/recipients" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "type": "recipients",
      "attributes": {
        "recipientName": "John Doe",
        "bankAccountCurrency": "GBP",
        "bankAccountCountry": "GB",
        "iban": "GB29NWBK60161331926819"
      }
    }
  }'
```

#### List Transfers
```bash
curl -X GET "${MONEYCORP_ENDPOINT}/accounts/{accountID}/transfers?page[size]=100" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Create Transfer
```bash
curl -X POST "${MONEYCORP_ENDPOINT}/accounts/{accountID}/transfers" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "data": {
      "type": "transfers",
      "attributes": {
        "sourceAccountId": "account_123",
        "destinationAccountId": "account_456",
        "amount": 100.00,
        "currency": "GBP"
      }
    }
  }'
```

#### Get Transfer
```bash
curl -X GET "${MONEYCORP_ENDPOINT}/accounts/{accountID}/transfers/{transferID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Create Payment (Payout)
```bash
curl -X POST "${MONEYCORP_ENDPOINT}/accounts/{accountID}/payments" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "data": {
      "type": "payments",
      "attributes": {
        "recipientId": "recipient_789",
        "amount": 100.00,
        "currency": "GBP",
        "reference": "Payment reference"
      }
    }
  }'
```

---

## 12. Plaid

**Location:** `internal/connectors/plugins/public/plaid/`

**Authentication:** API Key Headers (`PLAID-CLIENT-ID`, `PLAID-SECRET`)

**Base URLs:**
- Sandbox: `https://sandbox.plaid.com`
- Production: `https://production.plaid.com`

Uses official Plaid Go SDK.

### Configuration

```json
{
  "clientID": "your_client_id",              // Required - Plaid client ID
  "clientSecret": "your_client_secret",      // Required - Plaid client secret
  "isSandbox": true                          // Optional - Use sandbox environment (default: false)
}
```

**Page Size:** 100 (max: 500)

**Supported:** 18 languages, 20 country codes

### Endpoints

#### Create User
```bash
curl -X POST "https://sandbox.plaid.com/user/create" \
  -H "PLAID-CLIENT-ID: ${CLIENT_ID}" \
  -H "PLAID-SECRET: ${CLIENT_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "client_user_id": "user_123"
  }'
```

#### Create Link Token
```bash
curl -X POST "https://sandbox.plaid.com/link/token/create" \
  -H "PLAID-CLIENT-ID: ${CLIENT_ID}" \
  -H "PLAID-SECRET: ${CLIENT_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "user": { "client_user_id": "user_123" },
    "client_name": "My App",
    "products": ["transactions"],
    "country_codes": ["US"],
    "language": "en"
  }'
```

**Response:**
```json
{
  "link_token": "link-sandbox-abc123...",
  "expiration": "2024-01-01T00:00:00Z"
}
```

#### Get Link Token
```bash
curl -X POST "https://sandbox.plaid.com/link/token/get" \
  -H "PLAID-CLIENT-ID: ${CLIENT_ID}" \
  -H "PLAID-SECRET: ${CLIENT_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "link_token": "link-sandbox-abc123..."
  }'
```

#### Exchange Public Token
```bash
curl -X POST "https://sandbox.plaid.com/item/public_token/exchange" \
  -H "PLAID-CLIENT-ID: ${CLIENT_ID}" \
  -H "PLAID-SECRET: ${CLIENT_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "public_token": "public-sandbox-xyz789..."
  }'
```

**Response:**
```json
{
  "access_token": "access-sandbox-abc123...",
  "item_id": "item_123"
}
```

#### Get Accounts
```bash
curl -X POST "https://sandbox.plaid.com/accounts/get" \
  -H "PLAID-CLIENT-ID: ${CLIENT_ID}" \
  -H "PLAID-SECRET: ${CLIENT_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "access-sandbox-abc123..."
  }'
```

#### Sync Transactions
```bash
curl -X POST "https://sandbox.plaid.com/transactions/sync" \
  -H "PLAID-CLIENT-ID: ${CLIENT_ID}" \
  -H "PLAID-SECRET: ${CLIENT_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "access-sandbox-abc123...",
    "cursor": "optional_cursor_from_previous_sync"
  }'
```

#### Delete User
```bash
curl -X POST "https://sandbox.plaid.com/user/delete" \
  -H "PLAID-CLIENT-ID: ${CLIENT_ID}" \
  -H "PLAID-SECRET: ${CLIENT_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "client_user_id": "user_123"
  }'
```

#### Remove Item
```bash
curl -X POST "https://sandbox.plaid.com/item/remove" \
  -H "PLAID-CLIENT-ID: ${CLIENT_ID}" \
  -H "PLAID-SECRET: ${CLIENT_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "access-sandbox-abc123..."
  }'
```

---

## 13. Powens

**Location:** `internal/connectors/plugins/public/powens/`

**Type:** Open Banking Connector (User Linking)

**Authentication:** OAuth2

**Base URL:** Configurable

**Webview URL:** `https://webview.powens.com`

### Configuration

```json
{
  "clientID": "your_client_id",              // Required - Powens client ID
  "clientSecret": "your_client_secret",      // Required - Powens client secret
  "configurationToken": "your_config_token", // Required - Configuration token
  "domain": "your-domain.powens.com",        // Required - Powens domain
  "maxConnectionsPerLink": 5,                // Required - Max connections per link (min: 1)
  "endpoint": "https://api.powens.com"       // Required - API endpoint
}
```

**Page Size:** 100 (max: 1000)

### Endpoints

#### Create User
```bash
curl -X POST "${POWENS_ENDPOINT}/api/v1/user/create" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "client_user_id": "user_123"
  }'
```

#### Create Authorization Grant
```bash
curl -X POST "${POWENS_ENDPOINT}/api/v1/oauth/authorization-grant" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "powens_user_123",
    "scope": "accounts"
  }'
```

---

## 14. Qonto

**Location:** `internal/connectors/plugins/public/qonto/`

**Authentication:** HTTP Basic Auth (ClientID:APIKey), optional `X-Qonto-Staging-Token` header

**Base URL:** Configurable (default: `https://thirdparty.qonto.com`)

### Configuration

```json
{
  "clientID": "your_client_id",              // Required - Qonto client ID (organization slug)
  "apiKey": "your_api_key",                  // Required - Qonto API key
  "endpoint": "https://thirdparty.qonto.com", // Required - API endpoint (must be valid URL)
  "stagingToken": "staging_token",           // Optional - Staging environment token
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 100)

### Endpoints

#### Get Organization (includes accounts and balances)
```bash
curl -X GET "https://thirdparty.qonto.com/v2/organization" \
  -u "${CLIENT_ID}:${API_KEY}" \
  -H "X-Qonto-Staging-Token: ${STAGING_TOKEN}"
```

**Response:**
```json
{
  "organization": {
    "slug": "my-org",
    "bank_accounts": [
      {
        "slug": "main-account",
        "iban": "FR7630001007941234567890185",
        "bic": "QNTOFRP1XXX",
        "currency": "EUR",
        "balance": 10000.00,
        "authorized_balance": 9500.00
      }
    ]
  }
}
```

#### List Beneficiaries
```bash
curl -X GET "https://thirdparty.qonto.com/v2/organization/{orgSlug}/beneficiaries?iban=${ACCOUNT_IBAN}&per_page=100&page=1" \
  -u "${CLIENT_ID}:${API_KEY}"
```

#### Search Transactions
```bash
curl -X GET "https://thirdparty.qonto.com/v2/transactions?slug=${ACCOUNT_SLUG}&iban=${ACCOUNT_IBAN}&per_page=100&current_page=1&settled_at_from=${FROM_DATE}" \
  -u "${CLIENT_ID}:${API_KEY}"
```

**Response:**
```json
{
  "transactions": [
    {
      "transaction_id": "txn_123",
      "amount": -100.00,
      "currency": "EUR",
      "side": "debit",
      "status": "completed",
      "settled_at": "2024-01-15T10:30:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "total_pages": 5,
    "per_page": 100
  }
}
```

#### Create Transfer
```bash
curl -X POST "https://thirdparty.qonto.com/v2/transactions" \
  -u "${CLIENT_ID}:${API_KEY}" \
  -H "Content-Type: application/json" \
  -H "X-Qonto-Idempotency-Key: unique-key-123" \
  -d '{
    "debit_iban": "FR7630001007941234567890185",
    "credit_iban": "FR7630001007941234567890999",
    "amount": 100.00,
    "currency": "EUR",
    "reference": "Transfer reference"
  }'
```

#### Create Payment (External Transfer)
```bash
curl -X POST "https://thirdparty.qonto.com/v2/transactions/external" \
  -u "${CLIENT_ID}:${API_KEY}" \
  -H "Content-Type: application/json" \
  -H "X-Qonto-Idempotency-Key: unique-key-123" \
  -d '{
    "debit_iban": "FR7630001007941234567890185",
    "beneficiary_id": "benef_123",
    "amount": 100.00,
    "currency": "EUR",
    "reference": "Payment reference"
  }'
```

---

## 15. Stripe

**Location:** `internal/connectors/plugins/public/stripe/`

**Authentication:** Bearer Token (API Key)

**Base URL:** `https://api.stripe.com`

Uses official Stripe Go SDK.

### Configuration

```json
{
  "apiKey": "sk_test_...",                   // Required - Stripe API key (secret key)
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 100)

### Endpoints

#### Get Account
```bash
curl -X GET "https://api.stripe.com/v1/account" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}"
```

#### Get Balance
```bash
curl -X GET "https://api.stripe.com/v1/balance" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}"
```

**Response:**
```json
{
  "object": "balance",
  "available": [
    { "amount": 100000, "currency": "usd" }
  ],
  "pending": [
    { "amount": 5000, "currency": "usd" }
  ]
}
```

#### List Balance Transactions (Payments)
```bash
curl -X GET "https://api.stripe.com/v1/balance_transactions?limit=100&created[gte]=${CREATED_FROM}" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}"
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "id": "txn_123",
      "amount": 10000,
      "currency": "usd",
      "type": "charge",
      "status": "available",
      "created": 1704067200
    }
  ],
  "has_more": true
}
```

#### Create Transfer
```bash
curl -X POST "https://api.stripe.com/v1/transfers" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Idempotency-Key: unique-key-123" \
  -d "amount=10000&currency=usd&destination=acct_connected_123&description=Transfer%20description"
```

#### Reverse Transfer
```bash
curl -X POST "https://api.stripe.com/v1/transfers/{transferID}/reversals" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}" \
  -H "Idempotency-Key: unique-key-456"
```

#### Create Payout
```bash
curl -X POST "https://api.stripe.com/v1/payouts" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Idempotency-Key: unique-key-123" \
  -d "amount=10000&currency=usd&destination=ba_external_123&description=Payout%20description"
```

#### Create External Bank Account
```bash
curl -X POST "https://api.stripe.com/v1/accounts/{accountID}/external_accounts" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "external_account[object]=bank_account&external_account[country]=US&external_account[currency]=usd&external_account[routing_number]=110000000&external_account[account_number]=000123456789"
```

#### List External Bank Accounts
```bash
curl -X GET "https://api.stripe.com/v1/accounts/{accountID}/external_accounts?object=bank_account&limit=100" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}"
```

#### Create Webhook Endpoint
```bash
curl -X POST "https://api.stripe.com/v1/webhook_endpoints" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "url=https://your-webhook-url.com/webhooks/stripe&enabled_events[]=payment_intent.succeeded&enabled_events[]=payout.paid"
```

#### Delete Webhook Endpoint
```bash
curl -X DELETE "https://api.stripe.com/v1/webhook_endpoints/{webhookID}" \
  -H "Authorization: Bearer ${STRIPE_API_KEY}"
```

---

## 16. Tink

**Location:** `internal/connectors/plugins/public/tink/`

**Authentication:** OAuth2 Client Credentials Flow

**Base URL:** Configurable (e.g., `https://api.tink.com`)

**Webview URL:** `https://link.tink.com/1.0/transactions`

### Configuration

```json
{
  "clientID": "your_client_id",              // Required - Tink client ID
  "clientSecret": "your_client_secret",      // Required - Tink client secret
  "endpoint": "https://api.tink.com"         // Required - API endpoint
}
```

**Page Size:** 100 (max: 100)

**Supported:** 18 markets, 16 locales

### OAuth2 Token Request
```bash
curl -X POST "https://api.tink.com/api/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}&scope=accounts:read,transactions:read"
```

**Response:**
```json
{
  "access_token": "abc123...",
  "token_type": "bearer",
  "expires_in": 3600,
  "scope": "accounts:read,transactions:read"
}
```

### Endpoints

#### Create User
```bash
curl -X POST "https://api.tink.com/api/v1/user/create" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "external_user_id": "user_123",
    "market": "SE",
    "locale": "en_US"
  }'
```

#### Get Authorization Grant
```bash
curl -X POST "https://api.tink.com/api/v1/oauth/authorization-grant" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "tink_user_123",
    "scope": "accounts:read,transactions:read"
  }'
```

#### Delegate Authorization
```bash
curl -X POST "https://api.tink.com/api/v1/oauth/authorization-grant/delegate" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "auth_code_123",
    "scope": "accounts:read"
  }'
```

#### Get User Access Token
```bash
curl -X POST "https://api.tink.com/api/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=${AUTH_CODE}&client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}"
```

#### List Accounts
```bash
curl -X GET "https://api.tink.com/data/v2/accounts?pageSize=100" \
  -H "Authorization: Bearer ${USER_ACCESS_TOKEN}"
```

**Response:**
```json
{
  "accounts": [
    {
      "id": "account_123",
      "name": "Checking Account",
      "type": "CHECKING",
      "balances": {
        "booked": { "amount": { "value": { "unscaledValue": 100000, "scale": 2 }, "currencyCode": "SEK" } }
      }
    }
  ],
  "nextPageToken": "token_for_next_page"
}
```

#### Get Account
```bash
curl -X GET "https://api.tink.com/data/v2/accounts/{accountID}" \
  -H "Authorization: Bearer ${USER_ACCESS_TOKEN}"
```

#### List Transactions
```bash
curl -X GET "https://api.tink.com/data/v2/transactions?pageSize=100&bookedDateGte=${FROM_DATE}" \
  -H "Authorization: Bearer ${USER_ACCESS_TOKEN}"
```

#### Delete Credentials
```bash
curl -X DELETE "https://api.tink.com/api/v1/credentials/{credentialsID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

#### Delete User
```bash
curl -X POST "https://api.tink.com/api/v1/user/delete" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "external_user_id": "user_123"
  }'
```

#### Create Webhook
```bash
curl -X POST "https://api.tink.com/events/v2/webhook-endpoints" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-webhook-url.com/webhooks/tink",
    "enabledEvents": ["account:updated", "transactions:modified"]
  }'
```

#### Delete Webhook
```bash
curl -X DELETE "https://api.tink.com/events/v2/webhook-endpoints/{webhookID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

---

## 17. Wise

**Location:** `internal/connectors/plugins/public/wise/`

**Authentication:** Bearer Token in Authorization header

**Base URL:** `https://api.wise.com` (hardcoded)

### Configuration

```json
{
  "apiKey": "your_api_key",                  // Required - Wise API key
  "webhookPublicKey": "-----BEGIN PUBLIC KEY-----...", // Required - RSA public key (PEM format)
  "pollingPeriod": "2m"                      // Optional - Polling interval (default: 2m)
}
```

**Page Size:** 100 (max: 100)

**Note:** The `webhookPublicKey` must be a valid RSA public key in PEM format for webhook signature verification.

### Endpoints

#### List Profiles
```bash
curl -X GET "https://api.wise.com/v2/profiles" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

**Response:**
```json
[
  {
    "id": 12345,
    "type": "BUSINESS",
    "details": {
      "name": "My Business"
    }
  }
]
```

#### List Balances
```bash
curl -X GET "https://api.wise.com/v4/profiles/{profileID}/balances?types=STANDARD" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

**Response:**
```json
[
  {
    "id": 67890,
    "currency": "EUR",
    "amount": { "value": 1000.00, "currency": "EUR" },
    "type": "STANDARD"
  }
]
```

#### Get Balance
```bash
curl -X GET "https://api.wise.com/v4/profiles/{profileID}/balances/{balanceID}" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

#### List Recipient Accounts (Paginated)
```bash
curl -X GET "https://api.wise.com/v2/accounts?profile={profileID}&size=100&sort=id,asc" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

#### Get Recipient Account
```bash
curl -X GET "https://api.wise.com/v1/accounts/{accountID}" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

#### List Transfers
```bash
curl -X GET "https://api.wise.com/v1/transfers?profile={profileID}&limit=100&offset=0" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

#### Get Transfer
```bash
curl -X GET "https://api.wise.com/v1/transfers/{transferID}" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

#### Create Quote
```bash
curl -X POST "https://api.wise.com/v3/profiles/{profileID}/quotes" \
  -H "Authorization: Bearer ${WISE_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "sourceCurrency": "EUR",
    "targetCurrency": "GBP",
    "sourceAmount": 100.00,
    "payOut": "BALANCE"
  }'
```

#### Get Quote
```bash
curl -X GET "https://api.wise.com/v3/profiles/{profileID}/quotes/{quoteID}" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

#### Create Transfer
```bash
curl -X POST "https://api.wise.com/v1/transfers" \
  -H "Authorization: Bearer ${WISE_API_KEY}" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: unique-key-123" \
  -d '{
    "targetAccount": 12345678,
    "quoteUuid": "quote-uuid-here",
    "customerTransactionId": "unique-customer-id",
    "details": {
      "reference": "Transfer reference"
    }
  }'
```

#### Create Webhook Subscription
```bash
curl -X POST "https://api.wise.com/v3/profiles/{profileID}/subscriptions" \
  -H "Authorization: Bearer ${WISE_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Webhook",
    "delivery": {
      "version": "2.0.0",
      "url": "https://your-webhook-url.com/webhooks/wise"
    },
    "trigger_on": "transfers#state-change"
  }'
```

#### List Webhooks
```bash
curl -X GET "https://api.wise.com/v3/profiles/{profileID}/subscriptions" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

#### Delete Webhook
```bash
curl -X DELETE "https://api.wise.com/v3/profiles/{profileID}/subscriptions/{subscriptionID}" \
  -H "Authorization: Bearer ${WISE_API_KEY}"
```

---

## 18. EE (Enterprise) Connectors

The connectors above are the community-edition (CE) connectors under
`internal/connectors/plugins/public/`. The platform also ships **enterprise
edition (EE) connectors**, which live in a separate tree and are compiled only
when the `ee` build tag is enabled:

- **Location:** `ee/plugins/<connector>/`
- **Build tag:** `//go:build ee` (CE builds exclude them; see
  `internal/connectors/plugins/registry/enterprise_ce.go` vs `enterprise_ee.go`)
- **Registration:** `internal/connectors/plugins/registry/generated_ee.go`
- **Capabilities/config compilation:** `just compile-connector-capabilities`
  and `just compile-connector-configs` both scan `ee/plugins` in addition to the
  public tree.

| EE Connector   | Location                       | Notes |
|----------------|--------------------------------|-------|
| Banking Bridge | `ee/plugins/bankingbridge/`    | Accounts, balances, transactions |
| Bitstamp       | `ee/plugins/bitstamp/`         | Crypto exchange |
| Coinbase Prime | `ee/plugins/coinbaseprime/`    | Crypto prime brokerage |
| Fireblocks     | `ee/plugins/fireblocks/`       | Digital-asset custody; accounts, assets, blockchains, transactions |
| Routable       | `ee/plugins/routable/`         | Payables/AP |

Endpoint-level detail for each EE connector is intentionally left as a
follow-up: it will be filled in per connector as contract tests are added for
them. Building or testing EE connectors requires the `ee` tag (e.g.
`go test -tags ee,contract ./ee/plugins/<connector>/...`).

---

## Authentication Summary

| Connector      | Auth Method     | Headers/Params                                   |
|----------------|-----------------|--------------------------------------------------|
| Adyen          | API Key         | `X-API-Key`                                      |
| Atlar          | HTTP Basic Auth | `Authorization: Basic`                           |
| Banking Circle | mTLS + OAuth2   | Client Certificates + `Authorization: Bearer`    |
| Column         | HTTP Basic Auth | `Authorization: Basic` (`:APIKey`)               |
| Currency Cloud | OAuth2 + Token  | `X-Auth-Token`                                   |
| Dummypay       | None            | N/A                                              |
| Generic        | Depends         | Custom                                           |
| Increase       | Bearer Token    | `Authorization: Bearer`                          |
| Mangopay       | OAuth2          | `Authorization: Bearer`                          |
| Modulr         | HTTP Basic Auth | `Authorization: Basic`                           |
| Moneycorp      | OAuth2          | `Authorization: Bearer`                          |
| Plaid          | API Key Headers | `PLAID-CLIENT-ID`, `PLAID-SECRET`                |
| Powens         | OAuth2          | `Authorization: Bearer`                          |
| Qonto          | HTTP Basic Auth | `Authorization: Basic` + `X-Qonto-Staging-Token` |
| Stripe         | Bearer Token    | `Authorization: Bearer`                          |
| Tink           | OAuth2          | `Authorization: Bearer`                          |
| Wise           | Bearer Token    | `Authorization: Bearer`                          |

---

## Common Operations by Connector

| Connector      | Accounts | Balances | Transactions | Transfers    | Payouts | Bank Accounts | Webhooks |
|----------------|----------|----------|--------------|--------------|---------|---------------|----------|
| Adyen          | Merchant | -        | -            | -            | -       | -             | Yes      |
| Atlar          | Yes      | Yes      | Yes          | Yes (Credit) | -       | Yes           | -        |
| Banking Circle | Yes      | Yes      | -            | Yes          | Yes     | Yes           | -        |
| Column         | Yes      | Yes      | Yes          | Yes          | Yes     | Yes           | Yes      |
| Currency Cloud | Yes      | Yes      | Yes          | Yes          | Yes     | Yes           | -        |
| Increase       | Yes      | Yes      | Yes          | Yes          | Yes     | Yes           | Yes      |
| Mangopay       | Wallets  | Yes      | Yes          | -            | Yes     | Yes           | Yes      |
| Modulr         | Yes      | Yes      | Yes          | Yes          | Yes     | -             | -        |
| Moneycorp      | Yes      | Yes      | Yes          | Yes          | Yes     | Yes           | -        |
| Plaid          | Yes      | -        | Yes          | -            | -       | -             | -        |
| Qonto          | Yes      | Yes      | Yes          | Yes          | Yes     | -             | -        |
| Stripe         | Yes      | Yes      | Yes          | Yes          | Yes     | Yes           | Yes      |
| Tink           | Yes      | Yes      | Yes          | -            | -       | -             | Yes      |
| Wise           | Profiles | Yes      | Yes          | Yes          | -       | Yes           | Yes      |

---

## Contract Testing Recommendations

### Purpose

Contract tests verify that external PSP APIs still conform to our expected request/response structures. Since we're consumers of these APIs (not providers), we run tests against **real sandbox environments** to detect breaking changes early.

### 1. What to Test

**Response Structure Verification:**
- Required fields are present
- Data types match expectations (string, int, array, etc.)
- Nested object structures are correct
- Pagination formats are consistent

**Authentication Flows:**
- OAuth2 token endpoints return expected token structure
- API key authentication works as documented
- Token refresh mechanisms function correctly

**Idempotency Behavior:**
- Duplicate requests with same idempotency key return same response
- Different idempotency keys create new resources

**Error Response Formats:**
- Error responses contain expected fields (`error`, `message`, `code`, etc.)
- HTTP status codes match documentation

### 2. Sample Test Structure

```
tests/
├── contracts/
│   ├── adyen/
│   │   ├── adyen_test.go
│   │   └── testdata/
│   │       ├── list_merchant_accounts_response.json
│   │       └── webhook_response.json
│   ├── stripe/
│   │   ├── stripe_test.go
│   │   └── testdata/
│   │       ├── balance_response.json
│   │       └── payout_response.json
│   └── ...
└── schemas/
    ├── adyen_schemas.json
    ├── stripe_schemas.json
    └── ...
```

### 3. Example Contract Test (Go)

```go
//go:build contract

package contracts

func TestStripeBalanceContract(t *testing.T) {
    client := stripe.NewClient(os.Getenv("STRIPE_TEST_API_KEY"))

    balance, err := client.GetBalance()
    require.NoError(t, err)

    // Verify response structure
    assert.NotNil(t, balance.Available, "available field must be present")
    assert.NotNil(t, balance.Pending, "pending field must be present")

    // Verify data types
    for _, b := range balance.Available {
        assert.NotEmpty(t, b.Currency, "currency must not be empty")
        assert.IsType(t, int64(0), b.Amount, "amount must be int64")
    }
}

func TestStripeListBalanceTransactionsContract(t *testing.T) {
    client := stripe.NewClient(os.Getenv("STRIPE_TEST_API_KEY"))

    txns, err := client.ListBalanceTransactions(100, "")
    require.NoError(t, err)

    // Verify pagination structure
    assert.NotNil(t, txns.HasMore, "has_more field must be present")

    // Verify transaction structure (if any exist)
    if len(txns.Data) > 0 {
        txn := txns.Data[0]
        assert.NotEmpty(t, txn.ID, "id must not be empty")
        assert.NotEmpty(t, txn.Type, "type must not be empty")
        assert.NotEmpty(t, txn.Currency, "currency must not be empty")
    }
}
```

### 4. JSON Schema Validation

Use JSON schemas to validate response structures:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "StripeBalance",
  "type": "object",
  "required": ["object", "available", "pending"],
  "properties": {
    "object": { "type": "string", "const": "balance" },
    "available": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["amount", "currency"],
        "properties": {
          "amount": { "type": "integer" },
          "currency": { "type": "string", "minLength": 3, "maxLength": 3 }
        }
      }
    },
    "pending": {
      "type": "array",
      "items": { "$ref": "#/properties/available/items" }
    }
  }
}
```

### 5. Running Contract Tests

Contract tests live next to each connector's `client` package under the
`//go:build contract` tag, so they are excluded from `just tests` (which only
enables `-tags it`). Run them via the Justfile target:

```bash
# Run the contract tests for a connector (defaults to adyen).
# Requires that connector's contract credentials in the environment;
# without them the suite Skips rather than fails.
ADYEN_CONTRACT_API_KEY=...    \
ADYEN_CONTRACT_COMPANY_ID=... \
  just contract-tests adyen
```

Under the hood this runs:

```bash
go test -tags contract -count=1 ./internal/connectors/plugins/public/<connector>/...
```

Currently implemented for: **adyen** (first connector). The merchant-account
ordering assertion is pinned to an in-source `expectedMerchantIDs` constant in
`internal/connectors/plugins/public/adyen/client/contract_test.go` — update that
slice when the seeded sandbox data legitimately changes.

### 6. CI/CD Integration

Contract tests run on a daily schedule (not on every commit) via
[`.github/workflows/contract-tests.yml`](../.github/workflows/contract-tests.yml).
The workflow mirrors the main `Tests` job environment (Namespace runner, Nix
dev shell) and uses a per-connector matrix so adding a connector is a one-line
change. Credentials are injected from repository secrets
(`ADYEN_CONTRACT_API_KEY`, `ADYEN_CONTRACT_COMPANY_ID`). A failed run is a red
check; a guarded `if: failure()` step is reserved as the Slack/issue
notification extension point (no secret required until configured).

### 7. Handling Contract Failures

When a contract test fails:

1. **Check PSP changelog** - Look for announced API changes
2. **Verify sandbox status** - PSP sandbox might be down or degraded
3. **Update connector code** - If the API changed, update the client implementation
4. **Update contract tests** - Reflect the new expected structure

---

*Generated for Formance Payments contract testing initiative.*
