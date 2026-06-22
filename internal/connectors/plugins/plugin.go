package plugins

import "github.com/formancehq/payments/pkg/domain/models"

type Plugin struct {
	models.PSPPlugin
	models.OpenBankingPlugin
}
