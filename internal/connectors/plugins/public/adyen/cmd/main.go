package main

import (
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/adyen"
	"github.com/formancehq/payments/internal/models"
)

func main() {
	pluginFn := func() models.Plugin { return &adyen.Plugin{} }
	service.Execute(plugins.NewPlugin("adyen", pluginFn))
}