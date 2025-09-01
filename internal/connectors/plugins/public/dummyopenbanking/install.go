//go:build !it

package dummyopenbanking

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) install(_ context.Context, _ models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{}, plugins.ErrNotImplemented
}
