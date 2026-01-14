# Transaction

A payment transaction returned from the PSP.

## Properties

| Name | Type | Description | Required |
|------|------|-------------|----------|
| **Id** | **string** | Unique transaction identifier from the PSP | ✅ |
| **RelatedTransactionID** | Pointer to **string** | Parent transaction ID for adjustments (refunds, chargebacks). When set, this transaction becomes an adjustment of the parent. | |
| **CreatedAt** | **time.Time** | Transaction creation timestamp | ✅ |
| **UpdatedAt** | **time.Time** | Last update timestamp (used for pagination) | ✅ |
| **Currency** | **string** | Asset in UMN format (e.g., "USD/2", "BTC/8", "EUR/2") | ✅ |
| **Type** | [**TransactionType**](TransactionType.md) | Transaction type: PAYIN, PAYOUT, TRANSFER, OTHER | ✅ |
| **Status** | [**TransactionStatus**](TransactionStatus.md) | Transaction status | ✅ |
| **Amount** | **string** | Amount in minor units (integer string, e.g., "10000" for $100.00) | ✅ |
| **Scheme** | Pointer to **string** | Payment scheme (visa, mastercard, etc.) | |
| **SourceAccountID** | Pointer to **string** | Source account identifier | |
| **DestinationAccountID** | Pointer to **string** | Destination account identifier | |
| **Metadata** | Pointer to **map[string]string** | Additional key-value metadata | |

## Example: Standard Transaction

```json
{
  "id": "tx_abc123",
  "createdAt": "2026-01-14T10:00:00Z",
  "updatedAt": "2026-01-14T10:00:00Z",
  "currency": "USD/2",
  "type": "PAYIN",
  "status": "SUCCEEDED",
  "amount": "10000",
  "destinationAccountID": "acc_001",
  "metadata": {
    "orderId": "order-123"
  }
}
```

## Example: Refund (Payment Adjustment)

When returning a refund, set `relatedTransactionID` to link it to the original payment:

```json
{
  "id": "refund_xyz789",
  "relatedTransactionID": "tx_abc123",
  "createdAt": "2026-01-14T12:00:00Z",
  "updatedAt": "2026-01-14T12:00:00Z",
  "currency": "USD/2",
  "type": "OTHER",
  "status": "REFUNDED",
  "amount": "10000",
  "sourceAccountID": "acc_001",
  "metadata": {
    "reason": "customer_request"
  }
}
```

## Example: Chargeback

```json
{
  "id": "cb_dispute_001",
  "relatedTransactionID": "tx_abc123",
  "createdAt": "2026-01-14T14:00:00Z",
  "updatedAt": "2026-01-14T14:00:00Z",
  "currency": "USD/2",
  "type": "OTHER",
  "status": "DISPUTE",
  "amount": "10000"
}
```

## Pagination Notes

Transactions should be returned sorted by `updatedAt` ascending. Formance uses `updatedAtFrom` to fetch only transactions updated after the last sync.

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

