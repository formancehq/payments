package teller

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	// Teller Connect is a client-side JS widget. There is no server-side
	// user creation — this is a no-op.
	return models.CreateUserResponse{
		Metadata: map[string]string{},
	}, nil
}
