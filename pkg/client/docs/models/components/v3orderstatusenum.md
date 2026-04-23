# V3OrderStatusEnum

Lifecycle of an order on the exchange.
`PENDING` — accepted by the exchange, not yet working.
`OPEN` — live on the book, no fills yet.
`PARTIALLY_FILLED` — live on the book, some base quantity filled.
`FILLED` — fully filled, terminal.
`CANCELLED` — cancelled by the user or system, terminal.
`FAILED` — rejected by the exchange, terminal. See `error` for details.
`EXPIRED` — `timeInForce` elapsed before full fill, terminal.



## Values

| Name                               | Value                              |
| ---------------------------------- | ---------------------------------- |
| `V3OrderStatusEnumUnknown`         | UNKNOWN                            |
| `V3OrderStatusEnumPending`         | PENDING                            |
| `V3OrderStatusEnumOpen`            | OPEN                               |
| `V3OrderStatusEnumPartiallyFilled` | PARTIALLY_FILLED                   |
| `V3OrderStatusEnumFilled`          | FILLED                             |
| `V3OrderStatusEnumCancelled`       | CANCELLED                          |
| `V3OrderStatusEnumFailed`          | FAILED                             |
| `V3OrderStatusEnumExpired`         | EXPIRED                            |