package registry

import (
	"encoding/json"
	"errors"
	"fmt"
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
	if val.Kind() == reflect.Invalid {
		log.Panicf("RegisterPlugin config cannot be nil")
	}
	if val.Kind() != reflect.Struct {
		log.Panicf("RegisterPlugin config must be a struct, got %v", val.Kind())
	}
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		if !field.IsExported() {
			continue
		}

		validatorTag := field.Tag.Get("validate")

		jsonTag := field.Tag.Get("json")
		fieldName := strings.Split(jsonTag, ",")[0]

		if fieldName == "" || fieldName == "-" {
			continue
		}

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
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}

	p, err := info.createFunc(connectorName, logger, rawConfig)
	if err != nil {
		return nil, translateError(err)
	}

	return New(logger, p), nil
}

func GetCapabilities(provider string) ([]models.Capability, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
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
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.config, nil
}
