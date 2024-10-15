package main

import (
	"github.com/formancehq/go-libs/service"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/generic"
	"github.com/formancehq/payments/internal/models"
)

func main() {
	pluginFn := func() models.Plugin { return &generic.Plugin{} }
	service.Execute(plugins.NewPlugin("generic", pluginFn))
}
