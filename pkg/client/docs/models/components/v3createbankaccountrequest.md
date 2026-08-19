# V3CreateBankAccountRequest


## Fields

| Field                                                               | Type                                                                | Required                                                            | Description                                                         |
| ------------------------------------------------------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------- |
| `Name`                                                              | *string*                                                            | :heavy_check_mark:                                                  | Human-readable name for the bank account                            |
| `AccountNumber`                                                     | **string*                                                           | :heavy_minus_sign:                                                  | Domestic account number. Supply this or an IBAN                     |
| `Iban`                                                              | **string*                                                           | :heavy_minus_sign:                                                  | International bank account number. Supply this or an account number |
| `SwiftBicCode`                                                      | **string*                                                           | :heavy_minus_sign:                                                  | SWIFT/BIC code identifying the bank                                 |
| `Country`                                                           | **string*                                                           | :heavy_minus_sign:                                                  | Country the account is held in, as an ISO 3166-1 alpha-2 code       |
| `Metadata`                                                          | map[string]*string*                                                 | :heavy_minus_sign:                                                  | Arbitrary key/value pairs attached to the resource                  |