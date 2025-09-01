package main

type V3ConnectorConfigYaml struct {
	Components Components `yaml:"components"`
}

type Components struct {
	Schemas Schemas `yaml:"schemas"`
}

type Schemas struct {
	V3ConnectorConfig V3ConnectorConfig   `yaml:"V3ConnectorConfig"`
	V3Configs         map[string]V3Config `yaml:",inline"`
}

type Discriminator struct {
	PropertyName string            `yaml:"propertyName"`
	Mapping      map[string]string `yaml:"mapping"`
}

type V3ConnectorConfig struct {
	Discriminator Discriminator `yaml:"discriminator"`
	OneOf         []OneOf       `yaml:"oneOf"`
}

type OneOf struct {
	Ref map[string]string `yaml:",inline"`
}

type V3Config struct {
	Type       string              `yaml:"type"`
	Required   []string            `yaml:"required"`
	Properties map[string]Property `yaml:"properties"`
}

type Property struct {
	Type    string `yaml:"type"`
	Default any    `yaml:"default,omitempty"`
}

type ConfigJson map[string]ConfigProperties

type ConfigProperties struct {
	DataType     string `json:"dataType"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue"`
}
