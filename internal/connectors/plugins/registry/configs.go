package registry

type Type string

const (
	TypeLongString              Type = "long string"
	TypeString                  Type = "string"
	TypeDurationNs              Type = "duration ns"
	TypeDurationUnsignedInteger Type = "unsigned integer"
	TypeBoolean                 Type = "boolean"
)

type Configs map[string]Config
type Config map[string]Parameter
type Parameter struct {
	DataType     Type   `json:"dataType"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue"`
}

var (
	defaultParameters = map[string]Parameter{
		"pollingPeriod": {
			DataType:     "duration ns",
			Required:     false,
			DefaultValue: "2m",
		},
		"pageSize": {
			DataType:     "unsigned integer",
			Required:     false,
			DefaultValue: "100",
		},
		"name": {
			DataType: "string",
			Required: true,
		},
	}
)
