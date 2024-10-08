package api

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/go-libs/api"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/adyen"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/atlar"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/bankingcircle"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/configtemplate"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/currencycloud"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/dummypay"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/generic"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/mangopay"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/modulr"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/moneycorp"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/stripe"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/wise"
)

func connectorConfigsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: It's not ideal to re-identify available connectors
		// Refactor it when refactoring the HTTP lib.

		configs := configtemplate.BuildConfigs(
			atlar.Config{},
			adyen.Config{},
			bankingcircle.Config{},
			currencycloud.Config{},
			dummypay.Config{},
			modulr.Config{},
			stripe.Config{},
			wise.Config{},
			mangopay.Config{},
			moneycorp.Config{},
			generic.Config{},
		)

		err := json.NewEncoder(w).Encode(api.BaseResponse[configtemplate.Configs]{
			Data: &configs,
		})
		if err != nil {
			api.InternalServerError(w, r, err)
			return
		}
	}
}
