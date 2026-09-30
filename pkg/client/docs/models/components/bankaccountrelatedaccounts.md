# BankAccountRelatedAccounts


## Fields

| Field                                                                   | Type                                                                    | Required                                                                | Description                                                             |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `ID`                                                                    | *string*                                                                | :heavy_check_mark:                                                      | Unique identifier of the link between the bank account and the provider |
| `CreatedAt`                                                             | [time.Time](https://pkg.go.dev/time#Time)                               | :heavy_check_mark:                                                      | When the bank account was forwarded to this provider                    |
| `Provider`                                                              | *string*                                                                | :heavy_check_mark:                                                      | Name of the payment provider behind the connector                       |
| `ConnectorID`                                                           | *string*                                                                | :heavy_check_mark:                                                      | Identifier of the connector holding the provider-side account           |
| `AccountID`                                                             | *string*                                                                | :heavy_check_mark:                                                      | Identifier of the provider-side account                                 |