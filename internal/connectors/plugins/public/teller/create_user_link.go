package teller

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
)

type tellerConnectConfig struct {
	ApplicationID string `json:"applicationID"`
	Environment   string `json:"environment"`
}

func (p *Plugin) createUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	environment := "production"
	if p.config.IsSandbox {
		environment = "sandbox"
	}

	cfg := tellerConnectConfig{
		ApplicationID: p.config.ApplicationID,
		Environment:   environment,
	}

	linkPayload, err := json.Marshal(cfg)
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	// Teller Connect is a JS widget, not a redirect URL.
	// The Link field carries a JSON payload that the frontend uses to
	// initialize the Teller Connect widget.
	return models.CreateUserLinkResponse{
		Link: string(linkPayload),
	}, nil
}
