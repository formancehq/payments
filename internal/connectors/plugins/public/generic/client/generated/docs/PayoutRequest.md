# PayoutRequest

Request body for creating a payout (payment to external beneficiary).

## Properties

| Name | Type | Description | Required |
|------|------|-------------|----------|
| **IdempotencyKey** | **string** | Unique identifier for the payout request. Used for idempotency - if a payout with this key already exists, the existing payout is returned. | ✅ |
| **Amount** | **string** | Payout amount in minor units (integer string, e.g., "10000" for $100.00) | ✅ |
| **Currency** | **string** | Asset in UMN format (e.g., "USD/2", "BTC/8", "EUR/2") | ✅ |
| **SourceAccountId** | **string** | Source account identifier (internal account) | ✅ |
| **DestinationAccountId** | **string** | Destination account identifier (beneficiary/external account) | ✅ |
| **Description** | Pointer to **string** | Payout description/memo | |
| **Metadata** | Pointer to **map[string]string** | Additional key-value metadata | |

## Example

```json
{
  "idempotencyKey": "payout-123-abc",
  "amount": "10000",
  "currency": "USD/2",
  "sourceAccountId": "acc_internal_001",
  "destinationAccountId": "ben_external_002",
  "description": "Vendor payment",
  "metadata": {
    "invoiceId": "INV-2026-001"
  }
}
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

