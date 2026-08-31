# V3Pool

A named group of accounts whose balances are aggregated together


## Fields

| Field                                                                   | Type                                                                    | Required                                                                | Description                                                             |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `ID`                                                                    | *string*                                                                | :heavy_check_mark:                                                      | Unique identifier of the pool                                           |
| `Name`                                                                  | *string*                                                                | :heavy_check_mark:                                                      | Human-readable name of the pool                                         |
| `CreatedAt`                                                             | [time.Time](https://pkg.go.dev/time#Time)                               | :heavy_check_mark:                                                      | When the pool was created                                               |
| `Type`                                                                  | [*components.V3PoolTypeEnum](../../models/components/v3pooltypeenum.md) | :heavy_minus_sign:                                                      | Whether a pool holds a fixed account list or is driven by a query       |
| `Query`                                                                 | map[string]*any*                                                        | :heavy_minus_sign:                                                      | Filter selecting the accounts a dynamic pool contains                   |
| `PoolAccounts`                                                          | []*string*                                                              | :heavy_check_mark:                                                      | Accounts currently in the pool                                          |