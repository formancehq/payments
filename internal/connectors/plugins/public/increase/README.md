# Increase Connector

This connector enables synchronization of accounts, balances, payments, external accounts, and fund transfers via Increase's APIs. It basically integrates the Increase payment service provider with the Formance Payments service.

## Configuration

Required configuration parameters:
- `api_key`: Your Increase API key (required)
- `endpoint`: Your Increase endpoint (required)
- `webhook_shared_secret`: Your Increase webhook secret (required)

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
- Payment created webhooks
- Transfer created webhooks
- Payout created webhooks

## API Documentation

For detailed API documentation, refer to:
- [Increase API Documentation](https://increase.com/documentation/api)
