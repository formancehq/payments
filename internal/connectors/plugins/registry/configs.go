package registry

import (
	pkgregistry "github.com/formancehq/payments/pkg/registry"
)

// Type aliases for backward compatibility.
// These delegate to pkg/registry where the canonical types now live.
type Type = pkgregistry.Type

const (
	TypeLongString      = pkgregistry.TypeLongString
	TypeString          = pkgregistry.TypeString
	TypeDurationNs      = pkgregistry.TypeDurationNs
	TypeUnsignedInteger = pkgregistry.TypeUnsignedInteger
	TypeBoolean         = pkgregistry.TypeBoolean
)

// Configs, Config, and Parameter are aliases to pkg/registry types.
type Configs = pkgregistry.Configs
type Config = pkgregistry.Config
type Parameter = pkgregistry.Parameter
