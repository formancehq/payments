# Increase Connector

This connector integrates the Increase payment service provider with the Formance Payments service.

## Configuration

Required configuration parameters:
- `api_key`: Your Increase API key (required)
- `webhook_secret`: Secret for verifying webhook signatures (required)
- `polling_period`: Minimum interval between polling operations (default: 30s)

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

## Webhook Integration

Webhooks are automatically configured during connector installation. The connector supports:
- `account.created`: Account creation and updates
- `transaction.created`: Payment status updates
- `transfer.created`: Transfer status updates

### Security
- All webhooks are verified using HMAC signatures
- Invalid signatures are rejected
- Supports Increase's webhook replay protection

## Error Handling

### Rate Limits
- Maximum 100 requests per minute
- Minimum 30-second polling interval enforced
- Webhook processing is not rate-limited
- Implements exponential backoff for retries

### Common Errors
- `InvalidAPIKey`: Check your API key configuration
- `InvalidWebhookSignature`: Verify webhook secret configuration
- `AccountNotFound`: Ensure account exists and is accessible
- `InsufficientFunds`: Verify account balance before transfer
- `InvalidRoutingNumber`: Check bank account details
- `TransferLimitExceeded`: Review transfer amount and limits
- `MissingPayoutType`: Ensure the `payout_type` metadata key is set

## Known Limitations

1. API Rate Limits
   - Maximum 100 requests per minute
   - Webhook delivery may be delayed during high load
   - Polling operations respect minimum interval

2. Transfer Limits
   - ACH: Subject to daily/monthly limits
   - Wire: Subject to bank-specific limits
   - RTP: Maximum $100,000 per transfer
   - Reverse transfers not supported

3. Account Operations
   - Some operations require additional account verification
   - International accounts may have additional requirements
   - Real-time payment processing affected by rate limits

## API Documentation

For detailed API documentation, refer to:
- [Increase API Documentation](https://increase.com/documentation/api)
