package plaid

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

const (
	UserTokenMetadataKey = "user_token"
	LinkTokenMetadataKey = "link_token"
)

var (
	supportedLanguage = map[string]struct{}{
		"da": {},
		"nl": {},
		"en": {},
		"et": {},
		"fr": {},
		"de": {},
		"hi": {},
		"it": {},
		"lv": {},
		"lt": {},
		"no": {},
		"pl": {},
		"pt": {},
		"ro": {},
		"es": {},
		"sv": {},
		"vi": {},
	}

	supportedCountryCodes = map[string]struct{}{
		"AT": {},
		"BE": {},
		"CA": {},
		"DE": {},
		"DK": {},
		"EE": {},
		"ES": {},
		"FI": {},
		"FR": {},
		"GB": {},
		"IE": {},
		"IT": {},
		"LT": {},
		"LV": {},
		"NL": {},
		"NO": {},
		"PL": {},
		"PT": {},
		"SE": {},
		"US": {},
	}
)

type Config struct {
	ClientID     string `json:"clientID" validate:"required"`
	ClientSecret string `json:"clientSecret" validate:"required"`
	IsSandbox    bool   `json:"isSandbox" validate:""`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
