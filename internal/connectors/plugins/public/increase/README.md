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

## Configuration

Required configuration parameters:
- `apiKey`: Your Increase API key (required)
- `endpoint`: Your Increase endpoint (required)
- `webhookSharedSecret`: Your Increase webhook secret (required)

## API Documentation

For detailed API documentation, refer to:
- [Increase API Documentation](https://increase.com/documentation/api)
