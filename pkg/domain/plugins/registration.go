package plugins

import (
	"encoding/json"

	logging "github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/pkg/domain/models"
)

type CreateFunc func(
	models.ConnectorID,
	string,
	logging.Logger,
	json.RawMessage,
) (models.Plugin, error)

type Registration struct {
	PluginType   models.PluginType
	Capabilities []models.Capability
	CreateFunc   CreateFunc
	PageSize     uint64
	RawConf      any
}
