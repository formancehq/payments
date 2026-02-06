# Increase Connector

This connector enables synchronization of accounts, balances, payments, external accounts, and fund transfers via Increase's APIs. It basically integrates the Increase payment service provider with the Formance Payments service.

## Capabilities

### Account Operations
- Fetch accounts and balances

### Payment Operations
- Fetch payments
- Create transfers between accounts
- Create payouts using various methods:
  - ACH transfers
  - Wire transfers
  - Check transfers
  - Real-Time Payments Transfers

### External Account Operations
- Fetch external bank accounts  
- Create external bank accounts

### Webhooks Operations
- Account created webhooks
- External account created webhooks
- Payment created webhooks for different transaction types:
  - Succeeded transactions
  - Declined transactions
  - Pending transactions
- Account transfer created webhooks
- Payout created webhooks using various methods:
  - ACH transfer
  - Wire transfer
  - Check transfer

## Installation

```
curl -D - --data '{"name":"increase", "endpoint":"https://sandbox.increase.com", "apiKey":"", webhookSharedSecret: ""}' -X POST http://localhost:8080/v3/connectors/install/increase
```

### Configuration

Required configuration parameters:
- `apiKey`: Your Increase API key (required)
- `endpoint`: Your Increase endpoint (required)
- `webhookSharedSecret`: Your Increase webhook secret (required)

### Supported Currencies

The connector supports the following currencies:
  - CAD
  - CHF
  - EUR
  - GBP
  - JPY
  - USD

NOTE: Amounts are often represented in the smallest unit of the currency (e.g., cents for USD)

## Usage

### Creating Bank Accounts

Bank accounts requests are determined by some metadata in the request:
```json
{
  "name": "Gitstart Union Bank",
  "accountNumber": "09878762343",
  "metadata": {
    "com.increase.spec/description": "some description",
    "com.increase.spec/routingNumber": "2354321234",
    "com.increase.spec/accountHolder": "business",
  }
}
```

### Creating Payouts

Payout requests are determined by some metadata in the request.
Note: Payout destination id must be an account with name.


#### ACH

```json
{
  "metadata": {
    "com.increase.spec/payoutMethod": "ach",
  }
}
```

#### Wire

```json
{
  "metadata": {
    "com.increase.spec/payoutMethod": "wire",
  }
}
```

#### Check

```json
{
  "metadata": {
    "com.increase.spec/payoutMethod": "check",
    "com.increase.spec/sourceAccountNumberID": "account_number_zhlqj5dkyr95otox5nv3", // for check and rtp payout
    "com.increase.spec/fulfillmentMethod": "method", // third_party or physical_check
    "com.increase.spec/checkNumber": "789", //for third_party fulfillmentMethod
    "com.increase.spec/city": "some city", // for physical_check fulfillmentMethod
    "com.increase.spec/state": "some state", // for physical_check fulfillmentMethod
    "com.increase.spec/postalCode": "some postal code", // for physical_check fulfillmentMethod
    "com.increase.spec/line1": "some line", // for physical_check fulfillmentMethod
  }
}
```

#### RTP

```json
{
  "metadata": {
    "com.increase.spec/payoutMethod": "rtp",
    "com.increase.spec/sourceAccountNumberID": "account_number_zhlqj5dkyr95otox5nv3",
  }
}
```

### Webhooks

Webhook is supported for check, ach, wire, rtp, and account transfers
| Event name                            | Event URL Path                                          |
| ------------------------------------- | -------------------------------------------------- |
| pending_transaction.created           | /pending_transaction/created
| pending_transaction.updated           | /pending_transactions/updated
| declined_transaction.created          | /declined_transaction/created
| transaction.created                   | /transaction/created
| check_transfer.updated                | /check_transfer/updated

Note: To test the webhooks, you'd need to provide ```STACK_PUBLIC_URL``` to the payment worker env in docker file. You'd also need to add event name and url to the ```webhook_config``` table.

### Fetching Payments

Payments are fetched with their associated mandates to determine source and destination accounts.

#### Implementation Notes

1. **Pagination**:
   - Uses cursor-based pagination
   - Respects provided page size
   - Maintains state between requests
2. **Data Transformation**:
   - Converts Increase API responses to internal models
   - Handles timestamp conversions
   - Preserves all relevant user metadata

#### State Management
The connector maintains state for pagination and tracks the last processed item:
```json
{
  "NextCursor": "string", // Cursor for forward pagination
  "LastCreatedAt": "2024-01-01T00:00:00Z" // Timestamp of last processed item
}
```
##### How State Management Works

1. **Cursor-based Pagination**:
   - `NextCursor`: Points to the last item in the current page. Used to fetch the next page of results.
     ```json
     // Example: Current page ends with item "IN4123"
     { "NextCursor": "IN4123" } // Next request will start after "IN4123"

2. **Last Creation Date Tracking**:
   - `LastCreatedAt`: Timestamp of the most recently processed item
   - Prevents duplicate processing of items

   Example flow:
   ```json
   // Initial state
   {
    "LastCreatedAt": "2024-01-01T00:00:00Z"
   }
   // After processing items created on Jan 2nd
   {
    "LastCreatedAt": "2024-01-02T00:00:00Z",
    "NextCursor": "IN2456"
   }
   // Only items created after Jan 2nd will be processed in next fetch
   ```
3. **State Progression Example**:
   ```json
   // First request (no state)
   {
     "NextCursor": null,
     "LastCreatedAt": null
   }
   // After first page (50 items)
   {
     "NextCursor": "iuytrewqasdfghjk",
     "LastCreatedAt": "2024-01-01T12:00:00Z"
   }
   // After second page
   {
     "NextCursor": "876rdfghjklkjhgfd",
     "LastCreatedAt": "2024-01-01T14:00:00Z"
   }
   ```
This state management system ensures:
- No duplicate processing of items
- No missing items in case of failures
- Efficient pagination through large datasets
- Chronological ordering of processed items

## Error Handling

Common error scenarios:
1. Configuration Errors:
  - Missing required fields
3. Payout Creation:
  - Missing required metadata
  - Invalid currency
3. Bank Account Creation:
  - Missing required metadata
  - Invalid currency

## Testing

The connector includes comprehensive tests covering:
  - Configuration validation
  - Bank account creation
  - External account fetching
  - Account fetching
  - Payment fetching
  - Payout creation
  - Bank Account creation
  - Tranfer creation
  - Webhooks
  - Error scenarios

To run tests:
```bash
cd internal/connectors/plugins/public/increase
ginkgo -cover
```

### Sandbox Testing

For integration testing, use the Increase sandbox environment:
  1. Create a sandbox account at Increase
  2. Generate an API key
  3. Use the sandbox endpoint: `https://sandbox.increase.com`

## Implementation Details

The connector uses:
  - Stateful polling for data synchronization
  - Pagination handling for large datasets
  - Error mapping to standardized formats

## Support

For issues or questions:
  1. Check known limitations
  2. Verify configuration
  3. Check error handling documentation
  4. Consult Increase API documentation

## API Documentation

For detailed API documentation, refer to:
- [Increase API Documentation](https://increase.com/documentation/api)
