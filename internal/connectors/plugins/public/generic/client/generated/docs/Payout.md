# Payout

Response model for a payout (payment to external beneficiary).

## Properties

| Name | Type | Description | Required |
|------|------|-------------|----------|
| **Id** | **string** | PSP-generated unique payout identifier | ✅ |
| **IdempotencyKey** | **string** | Client-provided unique identifier for the payout request | ✅ |
| **Amount** | **string** | Payout amount in minor units (integer string) | ✅ |
| **Currency** | **string** | Asset in UMN format (e.g., "USD/2", "BTC/8") | ✅ |
| **SourceAccountId** | **string** | Source account identifier | ✅ |
| **DestinationAccountId** | **string** | Destination account identifier | ✅ |
| **Status** | [**TransactionStatus**](TransactionStatus.md) | Current payout status | ✅ |
| **CreatedAt** | **time.Time** | Payout creation timestamp | ✅ |
| **UpdatedAt** | Pointer to **time.Time** | Last update timestamp | |
| **Description** | Pointer to **string** | Payout description | |
| **Metadata** | Pointer to **map[string]string** | Additional metadata | |

## Example Response

```json
{
  "id": "payout_abc123",
  "idempotencyKey": "payout-123-abc",
  "amount": "10000",
  "currency": "USD/2",
  "sourceAccountId": "acc_internal_001",
  "destinationAccountId": "ben_external_002",
  "status": "PENDING",
  "createdAt": "2026-01-14T15:00:00Z",
  "updatedAt": "2026-01-14T15:00:00Z",
  "description": "Vendor payment",
  "metadata": {
    "invoiceId": "INV-2026-001"
  }
}
```

## Status Lifecycle

```
PENDING → PROCESSING → SUCCEEDED
                    ↘ FAILED
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

