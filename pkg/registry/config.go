package registry

// Type represents the data type of a configuration parameter.
type Type string

const (
	TypeLongString      Type = "long string"
	TypeString          Type = "string"
	TypeDurationNs      Type = "duration ns"
	TypeUnsignedInteger Type = "unsigned integer"
	TypeBoolean         Type = "boolean"
)

// Configs maps provider names to their configuration schemas.
type Configs map[string]Config

// Config maps parameter names to their definitions.
type Config map[string]Parameter

// Parameter defines a configuration parameter for a connector.
type Parameter struct {
	DataType     Type   `json:"dataType"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue"`
}

// defaultParameters are the common parameters that all connectors have.
var defaultParameters = map[string]Parameter{
	"pollingPeriod": {
		DataType:     TypeDurationNs,
		Required:     false,
		DefaultValue: "30m",
	},
	"name": {
		DataType: TypeString,
		Required: true,
	},
}
