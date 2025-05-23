package powens

import (
	"context"
	"strconv"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	createUserResponse, err := p.client.CreateUser(ctx)
	if err != nil {
		return models.CreateUserResponse{}, err
	}

	return models.CreateUserResponse{
		PermanentToken: &createUserResponse.AuthToken,
		Metadata: map[string]string{
			UserIDMetadataKey:    strconv.Itoa(createUserResponse.IdUser),
			ExpiresInMetadataKey: strconv.FormatInt(createUserResponse.ExpiresIn, 10),
		},
	}, nil
}
