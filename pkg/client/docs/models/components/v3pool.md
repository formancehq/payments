# V3Pool


## Fields

| Field                                                                  | Type                                                                   | Required                                                               | Description                                                            |
| ---------------------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `ID`                                                                   | *string*                                                               | :heavy_check_mark:                                                     | N/A                                                                    |
| `Name`                                                                 | *string*                                                               | :heavy_check_mark:                                                     | N/A                                                                    |
| `CreatedAt`                                                            | [time.Time](https://pkg.go.dev/time#Time)                              | :heavy_check_mark:                                                     | N/A                                                                    |
| `Type`                                                                 | [components.V3PoolTypeEnum](../../models/components/v3pooltypeenum.md) | :heavy_check_mark:                                                     | N/A                                                                    |
| `Query`                                                                | map[string]*any*                                                       | :heavy_minus_sign:                                                     | N/A                                                                    |
| `PoolAccounts`                                                         | []*string*                                                             | :heavy_check_mark:                                                     | N/A                                                                    |