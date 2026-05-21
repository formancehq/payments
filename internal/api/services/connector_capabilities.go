package services

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

func (s *Service) ConnectorsCapabilities() map[string][]models.Capability {
	return registry.GetAllCapabilities(s.debug)
}

func (s *Service) ConnectorsCapabilitiesGet(ctx context.Context, connectorID models.ConnectorID) ([]models.Capability, error) {
	connector, err := s.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return nil, newStorageError(err, "get connector")
	}

	caps, err := registry.GetCapabilities(connector.Provider)
	if err != nil {
		// Storage holds a row for a plugin no longer registered in this binary
		// (older build or feature flag turned off). Surface as 404 rather than
		// leaking the registry's internal error.
		if errors.Is(err, registry.ErrPluginNotFound) {
			return nil, fmt.Errorf("%w: %w", err, ErrNotFound)
		}
		return nil, err
	}
	return caps, nil
}
