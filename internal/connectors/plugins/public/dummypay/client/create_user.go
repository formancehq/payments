package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func (c *client) CreateUser(ctx context.Context, user models.PSPPaymentServiceUser) (string, error) {
	b, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("failed to marshal user: %w", err)
	}

	err = c.writeFile(fmt.Sprintf("user-%s.json", user.ID), b)
	if err != nil {
		return "", fmt.Errorf("failed to write user: %w", err)
	}

	return user.ID.String(), nil
}
