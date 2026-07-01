# TransferRequest

Request body for creating an internal transfer between accounts.

## Properties

| Name | Type | Description | Required |
|------|------|-------------|----------|
| **IdempotencyKey** | **string** | Unique identifier for the transfer request. Used for idempotency - if a transfer with this key already exists, the existing transfer is returned. | ✅ |
| **Amount** | **string** | Transfer amount in minor units (integer string, e.g., "10000" for $100.00) | ✅ |
| **Currency** | **string** | Asset in UMN format (e.g., "USD/2", "BTC/8", "EUR/2") | ✅ |
| **SourceAccountId** | **string** | Source account identifier (internal account) | ✅ |
| **DestinationAccountId** | **string** | Destination account identifier (internal account) | ✅ |
| **Description** | Pointer to **string** | Transfer description/memo | |
| **Metadata** | Pointer to **map[string]string** | Additional key-value metadata | |

## Example

```json
{
  "idempotencyKey": "transfer-456-def",
  "amount": "50000",
  "currency": "EUR/2",
  "sourceAccountId": "acc_main_001",
  "destinationAccountId": "acc_savings_002",
  "description": "Monthly savings transfer",
  "metadata": {
    "category": "savings"
  }
}
```

## Difference from Payout

- **Transfer**: Movement between two internal accounts (same PSP)
- **Payout**: Payment to an external beneficiary account

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

