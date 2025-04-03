# BankAccountRequest


## Fields

| Field               | Type                | Required            | Description         | Example             |
| ------------------- | ------------------- | ------------------- | ------------------- | ------------------- |
| `Country`           | *string*            | :heavy_check_mark:  | N/A                 | GB                  |
| `ConnectorID`       | *string*            | :heavy_check_mark:  | N/A                 |                     |
| `Name`              | *string*            | :heavy_check_mark:  | N/A                 | My account          |
| `AccountNumber`     | **string*           | :heavy_minus_sign:  | N/A                 |                     |
| `Iban`              | **string*           | :heavy_minus_sign:  | N/A                 |                     |
| `SwiftBicCode`      | **string*           | :heavy_minus_sign:  | N/A                 |                     |
| `Metadata`          | map[string]*string* | :heavy_minus_sign:  | N/A                 |                     |