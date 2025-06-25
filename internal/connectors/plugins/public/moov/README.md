# Moov Connector

The Moov connector integrates Formance with [Moov](https://moov.io), a modern payments infrastructure platform designed to simplify and streamline money movement for businesses and developers.

## Installation

To install the Column connector, use the following configuration:

```json
{
  "name": "moov",
  "pollingPeriod": "30s",
  "endpoint": "https://api.moov.io",
  "publicKey": "xxxxxxxxxxxxxxxxxxxx",
  "privateKey": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "accountID": "xxxxxxx-xxxxxx-xxxxxx-xxxx-xxxxxxxx",
  "pageSize": 1
}

```

### Configuration Parameters

| Parameter       | Description                                                                           | Required |
| --------------- | ------------------------------------------------------------------------------------- | -------- |
| `name`          | The name of the connector                                                             | Yes      |
| `pollingPeriod` | The frequency at which the connector polls for updates                                | Yes      |
| `endpoint`      | The Moov API endpoint                                                                 | Yes      |
| `privateKey`    | Your Moov API Private key                                                             | Yes      |
| `publicKey`     | Your Moov API Public key                                                              | Yes      |
| `accountID`     | Your Moov Merchant Account ID, can be gotten from business page in the moov dashboard | Yes      |
| `pageSize`      | The number of items to fetch per page                                                 | No       |

## Features

The Moov connector supports the following capabilities:

- Fetch accounts
- Fetch balances
- Fetch external accounts
- Fetch payments
- Create payout/transfers

## Fetching Payments

### Status Updates and Pagination

When working with Moov payments, it's important to understand these key behaviors:

1. **In-place Updates**: Moov updates payments in-place when their status changes, rather than creating new payment records. This means a single payment ID will represent the payment throughout its lifecycle, with its status field changing over time.

2. **Filtering Approach**: Due to this in-place update behavior, our connector:
   - Filters by status when fetching payments
   - Deliberately avoids using `endDateTime` filters, as they would limit results only to transfers up to the end of the day of the last transfer in the current page
   - Similarly avoids using `startDateTime` in some cases as it can cause similar pagination issues in reverse order

3. **Pagination Strategy**: To reliably capture all payments and their status changes:
   - The connector fetches payments for each status separately (Created, Pending, Completed, Failed, etc.)
   - Uses skip/count parameters for pagination
   - This approach ensures we don't miss payment status updates that would otherwise be filtered out

### ACH Settlement Times

**Important**: ACH payments typically take several hours to settle, and in some cases, can take 1-3 business days. During this time, the payment status will progress through various states:

- Initial statuses: `created` → `pending` → `queued`
- Final statuses: `completed` (success) or `failed` (rejection)

The connector polls periodically to capture these status changes.

### Implementation Notes

For implementers who need to track all payment updates:
- You may need to query each status type separately
- Consider implementing your own time-based filtering logic on top of the connector's results
- Monitor payment statuses by ID over time rather than expecting new records for status changes

### Payment Type Mapping

The connector determines payment types based on the presence of wallets and bank accounts in the transfer. Here's how Moov transfers are mapped to standard payment types:

1. **TRANSFER**: Assigned in two scenarios:
   - When both source and destination have wallet IDs (wallet-to-wallet transfer)
   - When both source and destination have bank accounts without wallets (direct bank-to-bank transfer)

2. **PAYOUT**: Assigned when only the source has a wallet ID
   - Represents money moving from a Moov wallet to an external destination (bank account, card, etc.)

3. **PAYIN**: Assigned when only the destination has a wallet ID
   - Represents money moving from an external source (bank account, card, etc.) to a Moov wallet

This mapping logic allows the connector to categorize Moov's various transfer types into standardized payment types that can be consistently used across the system.

## Creating Payment

The Moov connector handles payment creation by leveraging Moov's transfer capabilities while abstracting the complexities of payment method selection and validation.

### Payment Method Support

Moov supports various payment methods depending on the source and destination of funds:

#### Source Payment Methods
- **card-payment**: Credit/debit card payments to wallet
- **pull-from-card**: Pull funds from supported debit/prepaid cards
- **ach-debit-fund**: Fund from linked bank account
- **ach-debit-collect**: Pull funds for bill payments or direct debits
- **apple-pay**: Apple Pay transactions
- **moov-wallet**: Fund from Moov wallet

#### Destination Payment Methods
- **rtp-credit**: Real-time payments to bank accounts
- **push-to-card**: Push funds to debit/prepaid cards
- **ach-credit-standard**: Standard ACH to bank accounts
- **ach-credit-same-day**: Same-day ACH to bank accounts
- **moov-wallet**: Transfer to Moov wallet

### Transfer Options Validation

When creating a payment, the connector uses Moov's Transfer Options endpoint to validate if the selected payment methods are supported for the given source and destination. This validation ensures:

1. The payment methods are valid for the transaction type
2. The amount is within allowed limits for the chosen payment methods
3. The source and destination accounts have the required capabilities

The connector handles this validation internally by:
```go
// Internally, the connector validates the payment methods
transferOptions, err := c.service.GetMoovTransferOptions(ctx, 
    pr.Source.PaymentMethodID, 
    pr.Destination.PaymentMethodID, 
    pr.Amount.Value, 
    pr.Amount.Currency)
```

### Recommended Workflow

When integrating with the Moov connector, we recommend the following workflow:

1. **Fetch Account Information**:
   - Search accounts database to get the corresponding `sourceAccountId` and `destinationAccountId`

2. **Get Available Payment Methods**:
   - Use accounts IDs to fetch the supported payment methods between two accounts from Moov API. https://api.moov.io/transfer-options
   - Search the returned payment methods to get the corresponding `io.moov.spec/sourcePaymentMethodId` and `io.moov.spec/destinationPaymentMethodId`

3. **Create the Payment**:
   - Provide the source and destination payment method IDs
   - Include necessary metadata for the transaction (see Metadata Keys section)
   - Specify the transaction amount and currency

### Transfer Limits

Be aware of Moov's transfer limits when creating payments:
- **ach-debit-fund**: Up to $1,000,000 (same day) or $99,999,999.99 (standard)
- **push-to-card**: Up to $25,000
- **pull-from-card**: Up to $10,000
- **rtp-credit**: Up to $99,999,999.99

The connector will validate these limits when creating payments to prevent failed transactions.

Example of create payout payload
```json
{
  "amount": 1106, // in cents
  "reference": "mv_1032324323",
  "connectorID": "eyJQcm92aWRlciI6Im1vb......",
  "asset": "USD/2",
  "type": "PAYOUT",
  "description": "Another Salary payment for March 2025",
  "SourceAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vb3.........",
  "DestinationAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vb3YiL...........",
    "metadata": {
        "io.moov.spec/type": "ach",
        "io.moov.spec/destinationPaymentMethodId": "bb6160b7-8ecf-4f66-b72e-a9a96243689c",
        "io.moov.spec/sourcePaymentMethodId": "029ab065-82f0-4eb2-97c9-6d0bf363fd41"
    }
}
```

## Metadata Keys

The Moov connector uses the following metadata keys with the namespace `io.moov.spec/`:

### ACH Payment Creation

#### Source Payment Method Keys
- `sourcePaymentMethodId` - ID of the source payment method
- `sourcePaymentMethodType` - Type of source payment method (e.g., "ach-debit-fund")
- `sourceACHSecCode` - SEC code for ACH transactions (e.g., "CCD", "PPD", "WEB", "TEL")
- `sourceACHCompanyEntryDescription` - Company entry description for ACH transactions (max 10 characters)
- `sourceACHDebitHoldPeriod` - Hold period for ACH debits (e.g., "2-days")
- `sourceACHStatus` - Status of the ACH transaction (e.g., "initiated")
- `sourceACHTraceNumber` - Trace number for the ACH transaction
- `sourceACHInitiatedOn` - Timestamp when the ACH transaction was initiated

#### Destination Payment Method Keys
- `destinationPaymentMethodId` - ID of the destination payment method
- `destinationPaymentMethodType` - Type of destination payment method (e.g., "ach-credit-standard")
- `destinationACHCompanyEntryDescription` - Company entry description for destination ACH transactions (max 10 characters)
- `destinationACHOriginatingCompanyName` - Originating company name for ACH transactions (max 16 characters)

#### Bank Account Details
- `accountId` - The Moov account ID
- `bankName` - Name of the bank
- `holderType` - Type of account holder (e.g., "individual", "business")
- `bankAccountType` - Type of bank account (e.g., "checking", "savings")
- `routingNumber` - ABA routing number of the bank
- `lastFourAccountNumber` - Last four digits of the account number
- `fingerprint` - Unique fingerprint of the bank account
- `status` - Status of the bank account (e.g., "verified")

#### Source Bank Account-Specific Keys
- `sourceBankAccountId` - ID of the source bank account
- `sourceHolderName` - Name of the source account holder

#### Destination Bank Account-Specific Keys
- `destinationBankAccountId` - ID of the destination bank account
- `destinationHolderName` - Name of the destination account holder

#### Account Information
- `sourceAccountEmail` - Email address associated with the source account
- `sourceAccountDisplayName` - Display name of the source account
- `destinationAccountEmail` - Email address associated with the destination account
- `destinationAccountDisplayName` - Display name of the destination account

#### Fee Information
- `facilitatorFeeTotal` - Total facilitator fee amount
- `facilitatorFeeTotalDecimal` - Total facilitator fee amount as a decimal string
- `facilitatorFeeMarkup` - Markup amount for the facilitator fee
- `facilitatorFeeMarkupDecimal` - Markup amount for the facilitator fee as a decimal string

#### Sales Tax
- `salesTaxAmountCurrency` - Currency of the sales tax amount
- `salesTaxAmountvalue` - Value of the sales tax amount

#### Card Payment Keys
- `sourceCardDynamicDescriptor` - Dynamic descriptor for source card transactions (max 22 characters)
- `sourceCardTransactionSource` - Source of the card transaction (e.g., "first-recurring", "recurring", "unscheduled")
- `destinationCardDynamicDescriptor` - Dynamic descriptor for destination card transactions (max 22 characters)
