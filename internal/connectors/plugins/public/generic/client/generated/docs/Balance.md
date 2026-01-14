# Balance

An account balance for a specific currency/asset.

## Properties

| Name | Type | Description | Required |
|------|------|-------------|----------|
| **Amount** | **string** | Balance amount in minor units (integer string, e.g., "10000" for $100.00) | ✅ |
| **Currency** | **string** | Asset in UMN format (e.g., "USD/2", "BTC/8", "EUR/2") | ✅ |

## Example

```json
{
  "amount": "1000000",
  "currency": "USD/2"
}
```

This represents a balance of $10,000.00 USD.

## Multi-Currency Account Example

An account can have multiple balances in different currencies:

```json
{
  "id": "bal_123",
  "accountID": "acc_001",
  "at": "2026-01-14T15:00:00Z",
  "balances": [
    { "amount": "1000000", "currency": "USD/2" },
    { "amount": "500000", "currency": "EUR/2" },
    { "amount": "10000000", "currency": "BTC/8" }
  ]
}
```

## Amount Conversion

| Currency | Amount String | Human Readable |
|----------|---------------|----------------|
| USD/2 | "10000" | $100.00 |
| EUR/2 | "5000" | €50.00 |
| BTC/8 | "100000000" | 1.00000000 BTC |
| JPY/0 | "1000" | ¥1000 |

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

