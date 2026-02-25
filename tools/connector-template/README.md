# Connector's base code generation

## How to use

```sh
    go build ./
    sh connector-template.sh ../../internal/connectors/plugins/public <your_connector_name>
```

## Amount convention

All payment amounts (`PSPPayment.Amount`) must use the **gross** convention:
report the full amount as returned by the PSP **before** any fee deduction.
PSP fees should be stored in metadata (e.g. `fees`, `network_fees`) but never
subtracted from the amount field. This ensures consistent reconciliation
across connectors.