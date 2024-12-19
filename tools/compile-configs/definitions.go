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

type V3ConnectorConfig struct {
	AnyOf []AnyOf `yaml:"anyOf"`
}

type AnyOf struct {
	Ref map[string]string `yaml:",inline"`
}

type V3Config struct {
	Type       string              `yaml:"type"`
	Required   []string            `yaml:"required"`
	Properties map[string]Property `yaml:"properties"`
}

type Property struct {
	Type    string `yaml:"type"`
	Default string `yaml:"default,omitempty"`
}

type ConfigJson map[string]ConfigProperties

type ConfigProperties struct {
	DataType     string `json:"dataType"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue"`
}
