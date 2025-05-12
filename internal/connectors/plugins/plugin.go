package plugins

import "github.com/formancehq/payments/internal/models"

type Plugin struct {
	models.PSPPlugin
	models.BankingBridgePlugin
}
