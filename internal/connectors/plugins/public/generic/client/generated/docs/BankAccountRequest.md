# BankAccountRequest

Request body for creating a bank account (beneficiary/counterparty) that can be used as a destination for payouts.

## Properties

| Name | Type | Description | Required |
|------|------|-------------|----------|
| **Name** | **string** | Account holder name | ✅ |
| **AccountNumber** | Pointer to **string** | Bank account number | |
| **Iban** | Pointer to **string** | International Bank Account Number | |
| **SwiftBicCode** | Pointer to **string** | SWIFT/BIC code of the bank | |
| **Country** | Pointer to **string** | Country code (ISO 3166-1 alpha-2) | |
| **Metadata** | Pointer to **map[string]string** | Additional key-value metadata | |

## Example with IBAN

```json
{
  "name": "John Doe",
  "iban": "DE89370400440532013000",
  "swiftBicCode": "COBADEFFXXX",
  "country": "DE",
  "metadata": {
    "vendorId": "vendor-001"
  }
}
```

## Example with Account Number

```json
{
  "name": "Jane Smith",
  "accountNumber": "12345678",
  "swiftBicCode": "CHASUS33XXX",
  "country": "US",
  "metadata": {
    "type": "supplier"
  }
}
```

## Usage Notes

- At least one of `iban` or `accountNumber` should be provided
- `swiftBicCode` is recommended for international transfers
- The created bank account can be used as `destinationAccountId` in payout requests

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

