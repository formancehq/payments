package main

import (
	"github.com/formancehq/go-libs/service"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr"
	"github.com/formancehq/payments/internal/models"
)

func main() {
	pluginFn := func() models.Plugin { return &modulr.Plugin{} }
	service.Execute(plugins.NewPlugin("modulr", pluginFn))
}
