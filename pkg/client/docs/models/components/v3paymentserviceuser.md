# V3PaymentServiceUser

An end user on whose behalf payments and open banking connections are made


## Fields

| Field                                                                       | Type                                                                        | Required                                                                    | Description                                                                 |
| --------------------------------------------------------------------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `ID`                                                                        | *string*                                                                    | :heavy_check_mark:                                                          | Unique identifier of the payment service user                               |
| `Name`                                                                      | *string*                                                                    | :heavy_check_mark:                                                          | Full name of the payment service user                                       |
| `CreatedAt`                                                                 | [time.Time](https://pkg.go.dev/time#Time)                                   | :heavy_check_mark:                                                          | When the user was registered                                                |
| `ContactDetails`                                                            | [*components.V3ContactDetails](../../models/components/v3contactdetails.md) | :heavy_minus_sign:                                                          | How to reach a payment service user                                         |
| `Address`                                                                   | [*components.V3Address](../../models/components/v3address.md)               | :heavy_minus_sign:                                                          | A postal address                                                            |
| `BankAccountIDs`                                                            | []*string*                                                                  | :heavy_minus_sign:                                                          | Bank accounts associated with the user                                      |
| `Metadata`                                                                  | map[string]*string*                                                         | :heavy_minus_sign:                                                          | Arbitrary key/value pairs attached to the resource                          |