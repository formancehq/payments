package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
)

// DummyPSPName is the name of the dummy PSP used for testing.
const DummyPSPName = "dummypay"

// PluginCreateFunction is the factory function signature for creating plugins.
type PluginCreateFunction func(
	connector.ConnectorID,
	string,
	logging.Logger,
	json.RawMessage,
) (connector.Plugin, error)

// PluginInformation holds the metadata and factory for a registered plugin.
type PluginInformation struct {
	pluginType   connector.PluginType
	capabilities []connector.Capability
	createFunc   PluginCreateFunction
	config       Config
	pageSize     uint64
}

var (
	pluginsRegistry = make(map[string]PluginInformation)

	// ErrPluginNotFound is returned when a plugin is not found in the registry.
	ErrPluginNotFound = errors.New("plugin not found")

	checkRequired = regexp.MustCompile("required")
)

// RegisterPlugin registers a connector plugin with the global registry.
// This is typically called from an init() function in the connector package.
func RegisterPlugin(
	provider string,
	pluginType connector.PluginType,
	createFunc PluginCreateFunction,
	capabilities []connector.Capability,
	conf any,
	pageSize uint64,
) {
	pluginsRegistry[provider] = PluginInformation{
		pluginType:   pluginType,
		capabilities: capabilities,
		createFunc:   createFunc,
		config:       setupConfig(conf),
		pageSize:     pageSize,
	}
}

func setupConfig(conf any) Config {
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
			// Handle time.Duration and custom duration types like PollingPeriod
			typeName := field.Type.Name()
			if typeName == "Duration" || typeName == "PollingPeriod" {
				dataType = TypeDurationNs
				break
			}
			fallthrough
		case reflect.Bool:
			dataType = TypeBoolean
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

// GetPluginFactory returns the factory function and metadata for a plugin.
// This is used by internal code to create and wrap plugins.
func GetPluginFactory(provider string) (PluginCreateFunction, PluginInformation, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return nil, PluginInformation{}, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.createFunc, info, nil
}

// GetPluginType returns the plugin type for a provider.
func GetPluginType(provider string) (connector.PluginType, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return 0, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.pluginType, nil
}

// GetCapabilities returns the capabilities for a provider.
func GetCapabilities(provider string) ([]connector.Capability, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.capabilities, nil
}

// GetConfigs returns the configuration schemas for all registered providers.
func GetConfigs(debug bool) Configs {
	confs := make(Configs)
	for key, info := range pluginsRegistry {
		// hide dummy PSP outside of debug mode
		if !debug && key == DummyPSPName {
			continue
		}
		confs[key] = info.config
	}
	return confs
}

// GetConfig returns the configuration schema for a provider.
func GetConfig(provider string) (Config, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.config, nil
}

// GetPageSize returns the default page size for a provider.
func GetPageSize(provider string) (uint64, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return 0, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.pageSize, nil
}
