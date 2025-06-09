# GoCardless Connector

The GoCardless connector enables integration with the GoCardless payment service provider, allowing synchronization of creditors, customers, external accounts, and payments, as well as supporting bank account creation.

## Features

- Fetch creditors and customers
- Fetch external bank accounts (both creditor and customer accounts)
- Fetch payments with mandate support
- Create bank accounts for creditors and customers
- Stateful polling with pagination support

## Installation

### Configuration

The connector requires the following configuration:

```json
{
  "name": "gocardless",
  "pollingPeriod": "30s",
  "endpoint": "https://api-sandbox.gocardless.com",
  "shouldFetchMandate": "true", // this is indicating if mandate should be fetched when fetching payments and can only be "true" or "false" as string
  "accessToken": "****************************************", // GoCardless accessToken
  "pageSize": 50
}
```

| Field              | Required | Description                                         |
| ------------------ | -------- | --------------------------------------------------- |
| name               | Yes      | Name of the connector                               |
| pageSize           | Yes      | How many items to fetch per polling                 |
| shouldFetchMandate | Yes      | If mandate should be fetched when fetching payments |
| pollingPeriod      | Yes      | Interval at which polling happens                   |
| accessToken        | Yes      | GoCardless API access token                         |
| endpoint           | Yes      | GoCardless API endpoint (sandbox or production)     |

### Supported Currencies

The connector supports the following currencies:

- AUD
- CAD
- DKK
- EUR
- GBP
- NZD
- SEK
- USD

## Usage

### Creating Bank Accounts

Bank accounts can be created for both creditors and customers. The type is determined by metadata in the request:

```json
 { "name": "Gitstart Union Bank",
  "accountNumber": "21616901",
  "swiftBicCode": "021000021",
  "country": "US", // Country code in ISO 3166-1 alpha-2 code.
  "metadata": {
    "com.gocardless.spec/currency": "USD",
    "com.gocardless.spec/customer": "CU001DRQMY17P6",
    "com.gocardless.spec/account_type": "savings"
  }
}
{
  "name": "Gitstart Union Bank",
  "accountNumber": "21616901",
  "swiftBicCode": "021000021",
  "country": "US", // Country code in ISO 3166-1 alpha-2 code.
  "metadata": {
    "com.gocardless.spec/currency": "USD",
    "com.gocardless.spec/creditor": "CR123", // For creditor bank account
    // OR
    "com.gocardless.spec/customer": "CU123", // For customer bank account
    "com.gocardless.spec/account_type": "savings" // Required for US accounts
  }
}
```

### Fetching External Accounts

External accounts can be fetched for both creditors and customers:

```json
{
  "id": "CR123", // For creditor accounts
  // OR
  "id": "CU123" // For customer accounts
}
```

The id must start with:

- `CR` for creditor accounts
- `CU` for customer accounts

### Fetching Users (Creditors and Customers)

The connector supports fetching both creditors and customers through the `FetchNextOthers` endpoint. The type of user to fetch is determined by the id prefix in the request payload.

#### Fetching Creditors

```json
{
  "id": "CR123" // Must start with "CR" for creditors
}
```

#### Fetching Customers

```json
{
  "id": "CU123" // Must start with "CU" for customers
}
```

### Fetching Payments

Payments are fetched with their associated mandates to determine source and destination accounts.

#### Implementation Notes

1. **Pagination**:

   - Uses cursor-based pagination
   - Respects provided page size
   - Maintains state between requests

2. **Data Transformation**:

   - Converts GoCardless API responses to internal models
   - Handles timestamp conversions
   - Preserves all relevant user metadata

3. **Order**:
   - Items are returned in chronological order by creation date

#### State Management

The connector maintains state for pagination and tracks the last processed item:

```json
{
  "after": "string", // Cursor for forward pagination
  "before": "string", // Cursor for backward pagination
  "lastCreationDate": "2024-01-01T00:00:00Z" // Timestamp of last processed item
}
```

##### How State Management Works

1. **Cursor-based Pagination**:

   - `after`: Points to the last item in the current page. Used to fetch the next page of results.
     ```json
     // Example: Current page ends with item "CU123"
     { "after": "CU123" } // Next request will start after "CU123"
     ```
   - `before`: Points to the first item in the current page. Used for backwards pagination if needed.
     ```json
     // Example: Current page starts with item "CU789"
     { "before": "CU789" } // Previous page will end at "CU789"
     ```

2. **Last Creation Date Tracking**:

   - `lastCreationDate`: Timestamp of the most recently processed item
   - Prevents duplicate processing of items
   - Ensures chronological ordering

   Example flow:

   ```json
   // Initial state
   {
     "lastCreationDate": "2024-01-01T00:00:00Z"
   }

   // After processing items created on Jan 2nd
   {
     "lastCreationDate": "2024-01-02T00:00:00Z",
     "after": "CU456"
   }

   // Only items created after Jan 2nd will be processed in next fetch
   ```

3. **State Progression Example**:

   ```json
   // First request (no state)
   {
     "after": null,
     "before": null,
     "lastCreationDate": null
   }

   // After first page (50 items)
   {
     "after": "CU50",
     "before": "CU1",
     "lastCreationDate": "2024-01-01T12:00:00Z"
   }

   // After second page
   {
     "after": "CU100",
     "before": "CU51",
     "lastCreationDate": "2024-01-01T14:00:00Z"
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
   - Invalid access token
   - Invalid endpoint

2. Common errors when fetching users:

   - Invalid id format (must start with "CR" or "CU")
   - Missing id
   - API rate limits
   - Invalid pagination cursors

3. Bank Account Creation:

   - Missing required metadata
   - Invalid currency
   - Invalid account owner ID format
   - Invalid account type for US accounts

4. External Account Fetching:

   - Invalid id format
   - Non-existent account owner

## Known Limitations

1. Webhooks:

   - Currently not supported, Gocardless does not support creating webhook from API
   - Returns `ErrNotImplemented`

2. Transfers:

   - Creation and reversal not implemented
   - Returns `ErrNotImplemented`

3. Payouts:

   - Creation and reversal not implemented
   - Returns `ErrNotImplemented`

4. Balances:

   - Fetching of balances not implemented
   - Returns `ErrNotImplemented`

## Testing

The connector includes comprehensive tests covering:

- Configuration validation
- Bank account creation
- External account fetching
- Users fetching
- Payment fetching
- Error scenarios

To run tests:

```bash
cd internal/connectors/plugins/public/gocardless
ginkgo -cover
```

### Sandbox Testing

For integration testing, use the GoCardless sandbox environment:

1. Create a sandbox account at GoCardless
2. Generate an access token
3. Use the sandbox endpoint: `https://api-sandbox.gocardless.com`

## Implementation Details

The connector uses:

- GoCardless Pro SDK for Go
- Stateful polling for data synchronization
- Pagination handling for large datasets
- Error mapping to standardized formats

## Support

For issues or questions:

1. Check known limitations
2. Verify configuration
3. Check error handling documentation
4. Consult GoCardless API documentation

## References

- [GoCardless API Documentation](https://developer.gocardless.com/api-reference)
- [GoCardless Pro SDK for Go](https://github.com/gocardless/gocardless-pro-go)
