# Increase Connector

This connector integrates the Increase payment service provider with the Formance Payments service.

## Installation

1. Configure the connector with your Increase API key and desired polling period:
```json
{
    "apiKey": "your_api_key",
    "pollingPeriod": "30s"
}
```

Note: The minimum polling period is 30 seconds to respect API rate limits.

## Features

### Account Operations
- Fetch accounts and balances
- Monitor account status changes via webhooks

### Payment Operations
- Create transfers between accounts
- Create payouts using various methods:
  - ACH transfers (supports PPD and CCD)
  - Wire transfers
  - Check transfers
  - Real-Time Payments (RTP)

### External Account Operations
- Create and manage external bank accounts
- Monitor external account status changes

## Webhook Configuration

Webhooks must be configured in the Increase dashboard. The connector supports the following webhook events:
- `account.created`: Account creation and updates
- `transaction.created`: Payment status updates
- `transfer.created`: Transfer status updates

## Error Handling

### Rate Limits
- Increase enforces rate limits on API requests
- The connector implements a minimum 30-second polling interval
- Webhook processing is not rate-limited

### Common Errors
- Invalid API key: Check your configuration
- Missing payout type: Ensure the `payout_type` metadata key is set
- Invalid account status: Verify account is active
- Insufficient funds: Check account balance

## Known Limitations

1. Reverse transfers and payouts are not supported
2. Webhook configuration must be done manually in the Increase dashboard
3. Some operations may require additional account verification with Increase
4. API rate limits may affect real-time payment processing

## API Documentation

For detailed API documentation, refer to:
- [Increase API Documentation](https://increase.com/documentation/api)
