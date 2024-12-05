//go:build it

package dummypay

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) install(_ context.Context, _ models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}
