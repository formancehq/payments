package main

import (
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/wise"
	"github.com/formancehq/payments/internal/models"
)

func main() {
	pluginFn := func() models.Plugin { return &wise.Plugin{} }
	service.Execute(plugins.NewPlugin("wise", pluginFn))
}