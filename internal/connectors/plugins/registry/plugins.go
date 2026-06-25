package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"slices"
	"strings"

	logging "github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/pkg/domain/models"
	pkgplugins "github.com/formancehq/payments/pkg/domain/plugins"
)

const DummyPSPName = "dummypay"

// PluginCreateFunction is re-exported so callers referencing this type from
// the internal package continue to compile unmodified.
type PluginCreateFunction = pkgplugins.CreateFunc

var (
	ErrPluginNotFound       = errors.New("plugin not found")
	ErrPluginEnterpriseOnly = errors.New("connector is only available in the Enterprise Edition")

	checkRequired = regexp.MustCompile("required")
)

var pluginsRegistry map[string]pkgplugins.Registration

func load(registrations map[string]pkgplugins.Registration) {
	pluginsRegistry = registrations
}

// RegisterPlugin adds a single plugin to the registry.
// Used in tests to inject plugin doubles without going through the generated wiring files.
func RegisterPlugin(
	provider string,
	pluginType models.PluginType,
	createFunc PluginCreateFunction,
	capabilities []models.Capability,
	conf any,
	pageSize uint64,
) {
	if pluginsRegistry == nil {
		pluginsRegistry = make(map[string]pkgplugins.Registration)
	}
	pluginsRegistry[provider] = pkgplugins.Registration{
		PluginType:   pluginType,
		Capabilities: capabilities,
		CreateFunc:   createFunc,
		PageSize:     pageSize,
		RawConf:      conf,
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
			if field.Type.Name() == "Duration" {
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

// GetPlugin instantiates the named plugin and wraps it with the
// observability layer (tracing + structured logging). This function must
// stay in internal because the wrapper depends on internal/otel.
func GetPlugin(connectorID models.ConnectorID, logger logging.Logger, provider string, connectorName string, rawConfig json.RawMessage) (models.Plugin, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		if _, enterprise := EnterpriseOnlyPlugins[provider]; enterprise {
			return nil, fmt.Errorf("%s: %w", provider, ErrPluginEnterpriseOnly)
		}
		return nil, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}

	p, err := info.CreateFunc(connectorID, connectorName, logger, rawConfig)
	if err != nil {
		return nil, translateError(err)
	}

	return New(connectorID, logger, p), nil
}

func GetPluginType(provider string) (models.PluginType, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return 0, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.PluginType, nil
}

func GetCapabilities(provider string) ([]models.Capability, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.Capabilities, nil
}

// GetAllCapabilities mirrors GetConfigs: dummypay is the only PSP we expose
// solely to power debug/dev builds, so it must stay hidden from the public
// catalog. Each slice is cloned so callers cannot mutate the internal plugin
// registration through the returned map.
func GetAllCapabilities(debug bool) map[string][]models.Capability {
	caps := make(map[string][]models.Capability, len(pluginsRegistry))
	for key, info := range pluginsRegistry {
		if !debug && key == DummyPSPName {
			continue
		}
		caps[key] = slices.Clone(info.Capabilities)
	}
	return caps
}

func GetConfigs(debug bool) Configs {
	confs := make(Configs, len(pluginsRegistry))
	for key, info := range pluginsRegistry {
		if !debug && key == DummyPSPName {
			continue
		}
		confs[key] = setupConfig(info.RawConf)
	}
	return confs
}

func GetConfig(provider string) (Config, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return setupConfig(info.RawConf), nil
}

func GetPageSize(provider string) (uint64, error) {
	provider = strings.ToLower(provider)
	info, ok := pluginsRegistry[provider]
	if !ok {
		return 0, fmt.Errorf("%s: %w", provider, ErrPluginNotFound)
	}
	return info.PageSize, nil
}
