package main

import (
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/atlar"
	"github.com/formancehq/payments/internal/models"
)

func main() {
	pluginFn := func() models.Plugin { return &atlar.Plugin{} }
	service.Execute(plugins.NewPlugin("atlar", pluginFn))
}
