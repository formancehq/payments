package powens

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

const (
	UserIDMetadataKey    = "user_id"
	ExpiresInMetadataKey = "expires_in"

	powensWebviewBaseURL = "https://webview.powens.com"
)

const PAGE_SIZE = 100 // max page size is 1000

type Config struct {
	ClientID              string `json:"clientID" validate:"required"`
	ClientSecret          string `json:"clientSecret" validate:"required"`
	ConfigurationToken    string `json:"configurationToken" validate:"required"`
	Domain                string `json:"domain" validate:"required"`
	MaxConnectionsPerLink uint32 `json:"maxConnectionsPerLink" validate:"required,min=1"`
	Endpoint              string `json:"endpoint" validate:"required"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
