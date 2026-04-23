# V3TimeInForceEnum

How long an order is valid on the exchange.
`GOOD_UNTIL_CANCELLED` — rests until explicitly cancelled.
`GOOD_UNTIL_DATE_TIME` — rests until `expiresAt`.
`IMMEDIATE_OR_CANCEL` — fill immediately, cancel any unfilled portion.
`FILL_OR_KILL` — fill fully and immediately, or cancel entirely.



## Values

| Name                                  | Value                                 |
| ------------------------------------- | ------------------------------------- |
| `V3TimeInForceEnumUnknown`            | UNKNOWN                               |
| `V3TimeInForceEnumGoodUntilCancelled` | GOOD_UNTIL_CANCELLED                  |
| `V3TimeInForceEnumGoodUntilDateTime`  | GOOD_UNTIL_DATE_TIME                  |
| `V3TimeInForceEnumImmediateOrCancel`  | IMMEDIATE_OR_CANCEL                   |
| `V3TimeInForceEnumFillOrKill`         | FILL_OR_KILL                          |