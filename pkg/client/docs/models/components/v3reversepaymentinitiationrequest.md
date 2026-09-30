# V3ReversePaymentInitiationRequest


## Fields

| Field                                                                    | Type                                                                     | Required                                                                 | Description                                                              |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `Reference`                                                              | *string*                                                                 | :heavy_check_mark:                                                       | Caller-supplied identifier for the reversal, used to deduplicate retries |
| `Description`                                                            | *string*                                                                 | :heavy_check_mark:                                                       | Human-readable reason for the reversal                                   |
| `Amount`                                                                 | [*big.Int](https://pkg.go.dev/math/big#Int)                              | :heavy_check_mark:                                                       | Amount to reverse, in the asset's smallest unit                          |
| `Asset`                                                                  | *string*                                                                 | :heavy_check_mark:                                                       | Asset the reversal is denominated in                                     |
| `Metadata`                                                               | map[string]*string*                                                      | :heavy_minus_sign:                                                       | Arbitrary key/value pairs attached to the resource                       |