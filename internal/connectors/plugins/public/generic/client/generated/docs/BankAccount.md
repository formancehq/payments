# BankAccount

Response model for a bank account (beneficiary/counterparty).

## Properties

| Name | Type | Description | Required |
|------|------|-------------|----------|
| **Id** | **string** | PSP-generated unique bank account identifier | ✅ |
| **Name** | **string** | Account holder name | ✅ |
| **CreatedAt** | **time.Time** | Bank account creation timestamp | ✅ |
| **AccountNumber** | Pointer to **string** | Bank account number | |
| **Iban** | Pointer to **string** | International Bank Account Number | |
| **SwiftBicCode** | Pointer to **string** | SWIFT/BIC code of the bank | |
| **Country** | Pointer to **string** | Country code (ISO 3166-1 alpha-2) | |
| **Metadata** | Pointer to **map[string]string** | Additional metadata | |

## Example Response

```json
{
  "id": "ba_abc123xyz",
  "name": "John Doe",
  "createdAt": "2026-01-14T15:00:00Z",
  "iban": "DE89370400440532013000",
  "swiftBicCode": "COBADEFFXXX",
  "country": "DE",
  "metadata": {
    "vendorId": "vendor-001"
  }
}
```

## Usage

The returned `id` can be used as the `destinationAccountId` when creating payouts:

```json
{
  "idempotencyKey": "payout-001",
  "amount": "10000",
  "currency": "EUR/2",
  "sourceAccountId": "acc_internal_001",
  "destinationAccountId": "ba_abc123xyz",
  "description": "Payment to vendor"
}
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

