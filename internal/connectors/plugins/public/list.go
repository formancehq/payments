package public

import (
    _ "github.com/formancehq/payments/pkg/connectors/adyen"
    _ "github.com/formancehq/payments/pkg/connectors/atlar"
    _ "github.com/formancehq/payments/pkg/connectors/bankingcircle"
    _ "github.com/formancehq/payments/pkg/connectors/column"
    _ "github.com/formancehq/payments/pkg/connectors/currencycloud"
    _ "github.com/formancehq/payments/internal/connectors/plugins/public/dummypay"
    _ "github.com/formancehq/payments/internal/connectors/plugins/public/generic"
    _ "github.com/formancehq/payments/pkg/connectors/increase"
    _ "github.com/formancehq/payments/pkg/connectors/mangopay"
    _ "github.com/formancehq/payments/pkg/connectors/modulr"
    _ "github.com/formancehq/payments/pkg/connectors/moneycorp"
    _ "github.com/formancehq/payments/pkg/connectors/plaid"
    _ "github.com/formancehq/payments/pkg/connectors/powens"
    _ "github.com/formancehq/payments/pkg/connectors/qonto"
    _ "github.com/formancehq/payments/pkg/connectors/stripe"
    _ "github.com/formancehq/payments/pkg/connectors/tink"
    _ "github.com/formancehq/payments/pkg/connectors/wise"
)
