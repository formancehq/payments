# TransactionType

## Enum

Payment/transaction type. Core types are mapped to Formance payment types.

| Value | Description | Formance Type |
|-------|-------------|---------------|
| `PAYIN` | Incoming payment | `PAY-IN` |
| `PAYOUT` | Outgoing payment to external account | `PAYOUT` |
| `TRANSFER` | Internal transfer between accounts | `TRANSFER` |
| `OTHER` | Any other transaction type (refund, chargeback, etc.) | `OTHER` |

### Usage Notes

- **PAYIN**: Use for incoming funds (deposits, received payments)
- **PAYOUT**: Use for outgoing payments to external accounts (beneficiaries)
- **TRANSFER**: Use for internal movements between accounts within the PSP
- **OTHER**: Use for refunds, chargebacks, fees, adjustments, etc.

When using `OTHER`, you can specify the exact nature via the `status` field (e.g., `REFUNDED`, `DISPUTE`).

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

