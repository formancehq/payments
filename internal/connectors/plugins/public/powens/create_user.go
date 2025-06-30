package powens

import (
	"context"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	createUserResponse, err := p.client.CreateUser(ctx)
	if err != nil {
		return models.CreateUserResponse{}, err
	}

	expiresAt := time.Time{}
	if createUserResponse.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(createUserResponse.ExpiresIn) * time.Second)
	}

	return models.CreateUserResponse{
		PermanentToken: &models.Token{
			Token:     createUserResponse.AuthToken,
			ExpiresAt: expiresAt,
		},
		Metadata: map[string]string{
			UserIDMetadataKey:    strconv.Itoa(createUserResponse.IdUser),
			ExpiresInMetadataKey: strconv.FormatInt(createUserResponse.ExpiresIn, 10),
		},
	}, nil
}
