package main

import (
	"github.com/formancehq/go-libs/service"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe"
	"github.com/formancehq/payments/internal/models"
)

func main() {
	pluginFn := func() models.Plugin { return &stripe.Plugin{} }
	service.Execute(plugins.NewPlugin("stripe", pluginFn))
}
