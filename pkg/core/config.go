package core

type ConnectorConfigObject interface {
	Validate() error
}

type EmptyConnectorConfig struct{}

func (cfg EmptyConnectorConfig) Validate() error {
	return nil
}
