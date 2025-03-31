# AccountRequest


## Fields

| Field                                                            | Type                                                             | Required                                                         | Description                                                      |
| ---------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------- |
| `Reference`                                                      | *string*                                                         | :heavy_check_mark:                                               | N/A                                                              |
| `ConnectorID`                                                    | *string*                                                         | :heavy_check_mark:                                               | N/A                                                              |
| `CreatedAt`                                                      | [time.Time](https://pkg.go.dev/time#Time)                        | :heavy_check_mark:                                               | N/A                                                              |
| `Type`                                                           | [components.AccountType](../../models/components/accounttype.md) | :heavy_check_mark:                                               | N/A                                                              |
| `DefaultAsset`                                                   | **string*                                                        | :heavy_minus_sign:                                               | N/A                                                              |
| `AccountName`                                                    | **string*                                                        | :heavy_minus_sign:                                               | N/A                                                              |
| `Metadata`                                                       | map[string]*string*                                              | :heavy_minus_sign:                                               | N/A                                                              |