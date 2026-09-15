# V3ErrorResponse


## Fields

| Field                                                              | Type                                                               | Required                                                           | Description                                                        | Example                                                            |
| ------------------------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| `ErrorCode`                                                        | [components.V3ErrorsEnum](../../models/components/v3errorsenum.md) | :heavy_check_mark:                                                 | Machine-readable error code identifying the failure                | VALIDATION                                                         |
| `ErrorMessage`                                                     | *string*                                                           | :heavy_check_mark:                                                 | Human-readable description of the error                            | [VALIDATION] missing required config field: pollingPeriod          |
| `Details`                                                          | **string*                                                          | :heavy_minus_sign:                                                 | Optional link carrying additional context about the error          |                                                                    |