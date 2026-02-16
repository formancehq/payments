# Transfer

Response model for an internal transfer between accounts.

## Properties

| Name | Type | Description | Required |
|------|------|-------------|----------|
| **Id** | **string** | PSP-generated unique transfer identifier | Yes |
| **IdempotencyKey** | **string** | Client-provided unique identifier for the transfer request | Yes |
| **Amount** | **string** | Transfer amount in minor units (integer string) | Yes |
| **Currency** | **string** | Asset in UMN format (e.g., "USD/2", "BTC/8") | Yes |
| **SourceAccountId** | **string** | Source account identifier | Yes |
| **DestinationAccountId** | **string** | Destination account identifier | Yes |
| **Status** | [**TransactionStatus**](TransactionStatus.md) | Current transfer status | Yes |
| **CreatedAt** | **time.Time** | Transfer creation timestamp | Yes |
| **UpdatedAt** | Pointer to **time.Time** | Last update timestamp | |
| **Description** | Pointer to **string** | Transfer description | |
| **Metadata** | Pointer to **map[string]string** | Additional metadata | |

## Example Response

```json
{
  "id": "transfer_xyz789",
  "idempotencyKey": "transfer-456-def",
  "amount": "50000",
  "currency": "EUR/2",
  "sourceAccountId": "acc_main_001",
  "destinationAccountId": "acc_savings_002",
  "status": "SUCCEEDED",
  "createdAt": "2026-01-14T15:00:00Z",
  "updatedAt": "2026-01-14T15:01:00Z",
  "description": "Monthly savings transfer",
  "metadata": {
    "category": "savings"
  }
}
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
