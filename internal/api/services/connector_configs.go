package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
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

	var m map[string]interface{}
	err = json.Unmarshal(connector.Config, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal connector config: %w", err)
	}

	// inject provider into config json so SDK can distinguish between config types
	caser := cases.Title(language.English)
	m["provider"] = caser.String(connector.Provider)
	result, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connector config: %w", err)
	}

	rawConfig := json.RawMessage(result)
	return rawConfig, nil
}
