package registry

import (
	"encoding/json"
	"errors"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
)

type PluginCreateFunction func(string, logging.Logger, json.RawMessage) (models.Plugin, error)

type PluginInformation struct {
	capabilities []models.Capability
	createFunc   PluginCreateFunction
	config       Config
}

var (
	pluginsRegistry map[string]PluginInformation = make(map[string]PluginInformation)

	ErrPluginNotFound = errors.New("plugin not found")

	checkRequired = regexp.MustCompile("required")
)

func RegisterPlugin(
	provider string,
	createFunc PluginCreateFunction,
	capabilities []models.Capability,
	conf any,
) {
	pluginsRegistry[provider] = PluginInformation{
		capabilities: capabilities,
		createFunc:   createFunc,
		config:       setupConfig(provider, conf),
	}
}

func setupConfig(provider string, conf any) Config {
	config := make(Config)
	for paramName, param := range defaultParameters {
		if _, ok := config[paramName]; !ok {
			config[paramName] = param
		}
	}

	val := reflect.ValueOf(conf)
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		validatorTag := field.Tag.Get("validate")

		jsonTag := field.Tag.Get("json")
		fieldName := strings.Split(jsonTag, ",")[0]

		vt := field.Type
		var dataType Type
		switch vt.Kind() {
		case reflect.String:
			dataType = TypeString
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dataType = TypeUnsignedInteger
		case reflect.Int64:
			if field.Type.Name() == "Duration" {
				dataType = TypeDurationNs
				break
			}
			fallthrough
		default:
			log.Panicf("unhandled type for field %q: %q", val.Type().Field(i).Name, field.Type.Name())
		}

		config[fieldName] = Parameter{
			DataType: dataType,
			Required: checkRequired.MatchString(validatorTag),
		}
	}
	return config
}

func GetPlugin(logger logging.Logger, provider string, connectorName string, rawConfig json.RawMessage) (models.Plugin, error) {
	info, ok := pluginsRegistry[strings.ToLower(provider)]
	if !ok {
		return nil, ErrPluginNotFound
	}

	p, err := info.createFunc(connectorName, logger, rawConfig)
	if err != nil {
		return nil, translateError(err)
	}

	return New(logger, p), nil
}

func GetCapabilities(provider string) ([]models.Capability, error) {
	info, ok := pluginsRegistry[strings.ToLower(provider)]
	if !ok {
		return nil, ErrPluginNotFound
	}

	return info.capabilities, nil
}

func GetConfigs() Configs {
	confs := make(Configs)
	for key, info := range pluginsRegistry {
		confs[key] = info.config
	}
	return confs
}

func GetConfig(provider string) (Config, error) {
	info, ok := pluginsRegistry[strings.ToLower(provider)]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return info.config, nil
}
