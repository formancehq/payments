# TransactionStatus

## Enum

Payment/transaction status. Mapped to Formance payment statuses.

| Value | Description |
|-------|-------------|
| `PENDING` | Payment is awaiting processing |
| `SUCCEEDED` | Payment completed successfully |
| `FAILED` | Payment failed |
| `CANCELLED` | Payment was cancelled |
| `EXPIRED` | Payment expired |
| `REFUNDED` | Payment was refunded |
| `REFUNDED_FAILURE` | Refund attempt failed |
| `REFUND_REVERSED` | Refund was reversed |
| `DISPUTE` | Payment is under dispute |
| `DISPUTE_WON` | Dispute resolved in merchant's favor |
| `DISPUTE_LOST` | Dispute resolved against merchant |
| `AUTHORISATION` | Payment authorized but not captured |
| `CAPTURE` | Payment captured |
| `CAPTURE_FAILED` | Capture attempt failed |
| `OTHER` | Other/unknown status |

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
