package tink

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

const (
	UserIDMetadataKey = "user_id"
)

var (
	supportedMarkets = map[string]struct{}{
		"AT": {},
		"BE": {},
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
	}

	supportedLocales = map[string]struct{}{
		"en_US": {},
		"da_DK": {},
		"de_DE": {},
		"es_ES": {},
		"fi_FI": {},
		"fr_FR": {},
		"it_IT": {},
		"nl_NL": {},
		"no_NO": {},
		"pt_PT": {},
		"pl_PL": {},
		"sv_SE": {},
		"et_EE": {},
		"lt_LT": {},
		"lv_LV": {},
	}
)

type Config struct {
	ClientID     string `json:"clientID" validate:"required"`
	ClientSecret string `json:"clientSecret" validate:"required"`
	Endpoint     string `json:"endpoint" validate:"required"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
