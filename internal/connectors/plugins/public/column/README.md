# Column Connector

The Column connector integrates Formance with [Column](https://column.com), a modern banking-as-a-service platform that provides access to banking infrastructure through a simple API.

## Installation

To install the Column connector, use the following configuration:

```json
{
  "name": "column",
  "pollingPeriod": "30s",
  "endpoint": "https://api.column.com",
  "apiKey": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "pageSize": 1
}
```

### Configuration Parameters

| Parameter       | Description                                            | Required |
| --------------- | ------------------------------------------------------ | -------- |
| `name`          | The name of the connector                              | Yes      |
| `pollingPeriod` | The frequency at which the connector polls for updates | Yes      |
| `endpoint`      | The Column API endpoint                                | Yes      |
| `apiKey`        | Your Column API key                                    | Yes      |
| `pageSize`      | The number of items to fetch per page                  | No       |

## Features

The Column connector supports the following capabilities:

- Fetch accounts
- Fetch balances
- Fetch external accounts
- Fetch payments
- Create bank accounts
- Create transfers (internal transfers between accounts)
- Create payouts (external transfers to counterparties)
- Reverse payouts (for ACH transfers)
- Webhook integration

## Usage Examples

### Creating a Bank Account

To create a bank account, use the following payload:

```json
{
  "name": "Gitstart Union Bank",
  "accountNumber": "123457809012",
  "country": "US",
  "metadata": {
    "com.column.spec/routing_number": "021000021",
    "com.column.spec/routing_number_type": "aba",
    "com.column.spec/account_type": "checking",
    "com.column.spec/wire_drawdown_allowed": "false",
    "com.column.spec/address_line1": "123 Main Street",
    "com.column.spec/address_line2": "Apt 4B",
    "com.column.spec/city": "San Francisco",
    "com.column.spec/state": "CA",
    "com.column.spec/postal_code": "94105",
    "com.column.spec/email": "gitincunion@gitstart.com"
  }
}
```

### Internal Transfers

To create an internal transfer between Column accounts:

```json
{
  "amount": 50,
  "reference": "ba_407772345",
  "connectorID": "eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9",
  "asset": "USD/2",
  "type": "TRANSFER",
  "description": "Salary payment for March 2024",
  "DestinationAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJiYWNjXzJzdGMzRFRSbnpNbmszSmxHOUVhUDhXVDlpMSJ9",
  "SourceAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJiYWNjXzJzdGM2U3Jrd3VGdHFhZTBKbmtGcTVJbHA4NyJ9",
  "metadata": { // metadata is optional
    "com.column.spec/hold": "false",
    "com.column.spec/allow_overdraft": "false",
  }
}
```

### External Payouts

The Column connector supports several types of external payouts:

#### ACH Payout

```json
{
  "amount": 50,
  "reference": "ba_271422875",
  "connectorID": "eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9",
  "asset": "USD/2",
  "type": "PAYOUT",
  "description": "Salary payment for March 2024",
  "DestinationAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJjcHR5XzJ0ekpuTGhmNXlERk1vRnVmWEdGOFplSWp6YiJ9",
  "SourceAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJiYWNjXzJzdGMzRFRSbnpNbmszSmxHOUVhUDhXVDlpMSJ9",
  "metadata": {
    "com.column.spec/payout_type": "ach",
    "com.column.spec/type": "DEBIT",
    "com.column.spec/entry_class_code": "PPD"
  }
}
```

#### Wire Payout

```json
{
  "amount": 50,
  "reference": "ba_987622000",
  "connectorID": "eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9",
  "asset": "USD/2",
  "type": "PAYOUT",
  "description": "Salary payment for March 2024",
  "DestinationAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJjcHR5XzJ0ekpuTGhmNXlERk1vRnVmWEdGOFplSWp6YiJ9",
  "SourceAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJiYWNjXzJzdGMzRFRSbnpNbmszSmxHOUVhUDhXVDlpMSJ9",
  "metadata": {
    "com.column.spec/payout_type": "wire"
  }
}
```

#### International Wire Payout

```json
{
  "amount": 500,
  "reference": "ba_987651480",
  "connectorID": "eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9",
  "type": "PAYOUT",
  "asset": "USD/2",
  "description": "International supplier payment",
  "DestinationAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJjcHR5XzJ0VXdva0ZjSGdEY3VDWjlnbkNYaVdjVTdmbCJ9",
  "SourceAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJiYWNjXzJzdGMzRFRSbnpNbmszSmxHOUVhUDhXVDlpMSJ9",
  "metadata": {
    "com.column.spec/allow_overdraft": "true",
    "com.column.spec/charge_bearer": "SHAR",
    "com.column.spec/general_info": "March 2024 inventory payment",
    "com.column.spec/beneficiary_reference": "SUPPLIER-REF-001",
    "com.column.spec/payout_type": "international-wire"
  }
}
```

#### Real-time Payout

```json
{
  "amount": 50,
  "reference": "ba_982751800",
  "connectorID": "eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9",
  "type": "PAYOUT",
  "asset": "USD/2",
  "description": "International Supplier payment (Real time)",
  "DestinationAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJjcHR5XzJ0ekpuTGhmNXlERk1vRnVmWEdGOFplSWp6YiJ9",
  "SourceAccountID": "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9LCJSZWZlcmVuY2UiOiJiYWNjXzJzdGMzRFRSbnpNbmszSmxHOUVhUDhXVDlpMSJ9",
  "metadata": {
    "com.column.spec/allow_overdraft": "true",
    "com.column.spec/payout_type": "realtime"
  }
}
```

### Reversing ACH Payouts

Only ACH transactions can be reversed. To reverse an ACH payout:

1. First, create an ACH transfer and note the PaymentInitiationID from the response:

```json
{
  "description": "Salary payment for March 2024",
  "amount": 50,
  "bank_account_id": "bacc_2stc3DTRnzMnk3JlG9EaP8WT9i1",
  "counterparty_id": "cpty_2tzJnLhf5yDFMoFufXGF8ZeIjzb",
  "currency_code": "USD",
  "type": "DEBIT",
  "entry_class_code": "PPD"
}
```

2. Then, use the PaymentInitiationID to create a reversal:

```json
POST /v3/payment-initiations/:paymentInitiationID/reverse
{
    "description": "Salary payment for March 2024",
    "asset": "USD/2",
    "Reference": "Anything",
    "amount": 50,
    "metadata": {
        "com.column.spec/reason": "incorrect_amount"
    }
}
```

Valid reasons for reversal include:

- `duplicated_entry`
- `incorrect_amount`
- `incorrect_receiver_account`
- `debit_earlier_than_intended`
- `credit_later_than_intended`

## Webhook Integration

The Column connector supports webhook integration to receive real-time updates about transaction statuses.

### Setting Up Webhooks

1. Add the following environment variable to the payments_worker service in your docker-compose file:

   ```
   STACK_PUBLIC_URL: <Your edge URL>
   ```

2. Configure webhook endpoints in the `webhooks_config` table in your database:

   | ID  | Name                                     | ConnectorID   | URLPath                                   |
   | --- | ---------------------------------------- | ------------- | ----------------------------------------- |
   | 0   | book.transfer.completed                    | <connectorID> | /book/transfer/completed                    |
   | 1   | wire.outgoing_transfer.completed         | <connectorID> | /wire/outgoing_transfer/completed         |
   | 2   | ach.outgoing_transfer.settled            | <connectorID> | /ach/outgoing_transfer/settled            |
   | 3   | swift.outgoing_transfer.completed        | <connectorID> | /swift/outgoing_transfer/completed        |
   | 4   | realtime.outgoing_transfer.completed     | <connectorID> | /realtime/outgoing_transfer/completed     |

### Testing Webhooks

1. Get the Reference ID (e.g., `acht_2u0NyJbNrEkAFDYPhgWchRfXwHW`) from the workflow. The status should be `INITIATED`.

2. Settle the ACH payment:

   ```
   POST {{base_url}}/simulate/transfers/ach/settle
   {
       "ach_transfer_id": "acht_2u0NyJbNrEkAFDYPhgWchRfXwHW"
   }
   ```

3. Get the Column webhook endpoints:

   ```
   GET https://api.column.com/webhook-endpoints
   ```

4. For the event you want to test, get the ID for that URL and send a webhook verify event:

   ```
   POST https://api.column.com/webhook-endpoints/whep_2txFEjfex72J2VkHEXlaW1ZlkfT/verify
   {
       "event_type": "ach.outgoing_transfer.settled"
   }
   ```

5. From the summary in the ngrok dashboard, get the request body and Column Signature header.

6. Run the webhook:

   ```
   POST http://localhost:8080/v3/connectors/webhooks/eyJQcm92aWRlciI6ImNvbHVtbiIsIlJlZmVyZW5jZSI6IjkwNmEzODkyLTQyZmQtNDhkYi1iMGZlLWRhOTQwOTEwZmVmMSJ9/ach/outgoing_transfer/settled
   ```

   Include the Column Signature header and use the request body from the ngrok dashboard.

7. In the workflow, search for `RunHandleWebhooks` to ensure it completed successfully.

## Metadata Keys

The Column connector uses the following metadata keys with the namespace `com.column.spec/`:

### Bank Account Creation

- `routing_number` - The routing number of the bank
- `routing_number_type` - The type of routing number (e.g., "aba")
- `account_type` - The type of account (e.g., "checking")
- `wire_drawdown_allowed` - Whether wire drawdown is allowed ("true" or "false")
- `address_line1` - First line of the address
- `address_line2` - Second line of the address
- `city` - City
- `state` - State
- `postal_code` - Postal code
- `email` - Email address
- `phone` - Phone number
- `legal_id` - Legal ID
- `legal_type` - Legal type
- `local_bank_code` - Local bank code
- `local_account_number` - Local account number

### Payouts

- `payout_type` - Type of payout ("ach", "wire", "international-wire", "realtime")
- `allow_overdraft` - Whether to allow overdraft ("true" or "false")
- `type` - Type of ACH transfer ("DEBIT" or "CREDIT")
- `entry_class_code` - Entry class code for ACH transfers (e.g., "PPD")
- `charge_bearer` - Charge bearer for international wires (e.g., "SHAR")
- `general_info` - General information for international wires
- `beneficiary_reference` - Beneficiary reference for international wires

### Reversal

- `reason` - Reason for reversal (one of the valid reversal reasons)
