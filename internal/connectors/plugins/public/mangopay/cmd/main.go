package main

import (
	"github.com/formancehq/go-libs/service"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay"
	"github.com/formancehq/payments/internal/models"
)

func main() {
	pluginFn := func() models.Plugin { return &mangopay.Plugin{} }
	service.Execute(plugins.NewPlugin("mangopay", pluginFn))
}
