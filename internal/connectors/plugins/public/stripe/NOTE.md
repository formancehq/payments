# Stripe update: From polling to webhooks

For greater performances on the stripe connector, and to remove strain on payments-worker,
we decided to use stripe webhooks instead of pollings.
There will still be pollings to ensure no data has been lost, but it will be very sparse compared than today.

## Installation


No additional properties are needed for the required configurations, webhooks are created at the installation with the
correct events selected.
Webhook secret and id are then added to the table "webhooks config" after the creation.

Stripe allows to create multiple webhook endpoint and dispatch events as we please to different endpoints.

I do not see major pros (easy to debug by checking endpoint?) to split events between different endpoints, so this implementation routes every events
into a single endpoint.

## Uninstallation

If you uninstall the connector, the webhook endpoint will be deleted.

## Schedule



## Migration

If stack has a stripe connector, then we must update it, we should do the following steps:

### Setup webhooks

run createWebhook on it

### Modify schedule

The schedules should be updated.

### Update polling period in config?

If we chose to still keep config.pollingPeriod to determine the periodical "ensure data integrity",
we must modify the config and put a high value.

If we chose to enforce a default value and not use config.pollingPeriod, then we can either do nothing or remove it from the config?

## Issues

### Limited sandbox webhooks

Only 16 webhooksEndpoints are allowed in a sandbox. 

### Raw data incoherence

RawData would not be the same depending whether we got the information from the api or from the webhook

### CreatedAt incoherence

```
balanceTransaction.Created and balanceTransaction.Source.Payout.Created are generally not the same timestamps, although they're typically close to each other.
Here's the difference between these two timestamps:
    balanceTransaction.Created - This is when the balance transaction (the ledger entry) was created in your Stripe account. For a payout transaction, this represents when the movement of funds was recorded in your balance.
    balanceTransaction.Source.Payout.Created - This is when the payout object itself was created. This typically happens when Stripe initiates the payout process.
```

