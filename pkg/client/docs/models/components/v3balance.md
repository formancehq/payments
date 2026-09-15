# V3Balance


## Fields

| Field                                                 | Type                                                  | Required                                              | Description                                           |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `AccountID`                                           | *string*                                              | :heavy_check_mark:                                    | Identifier of the account this balance belongs to     |
| `CreatedAt`                                           | [time.Time](https://pkg.go.dev/time#Time)             | :heavy_check_mark:                                    | Start of the period this balance covers               |
| `LastUpdatedAt`                                       | [time.Time](https://pkg.go.dev/time#Time)             | :heavy_check_mark:                                    | When the balance was last refreshed from the provider |
| `Asset`                                               | *string*                                              | :heavy_check_mark:                                    | Asset the balance is denominated in                   |
| `Balance`                                             | [*big.Int](https://pkg.go.dev/math/big#Int)           | :heavy_check_mark:                                    | Amount held, in the asset's smallest unit             |