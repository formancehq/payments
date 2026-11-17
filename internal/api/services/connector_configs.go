package services

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/connectorconfig"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/pkg/domain/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func (s *Service) ConnectorsConfigs() registry.Configs {
	return registry.GetConfigs(s.debug)
}

func (s *Service) ConnectorsConfig(ctx context.Context, connectorID models.ConnectorID) (json.RawMessage, error) {
	connector, err := s.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return nil, newStorageError(err, "get connector")
	}

	var m map[string]string
	_ = json.Unmarshal(connectorconfig.Readable(connector.Config), &m)

	// Preserve the provider name used by existing configuration readers.
	caser := cases.Title(language.English)
	m["provider"] = caser.String(connector.Provider)
	result, _ := json.Marshal(m)
	return json.RawMessage(result), nil
}
