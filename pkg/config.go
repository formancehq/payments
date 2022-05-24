package payments

type ConnectorConfigObject interface {
	Validate() error
}

type EmptyConnectorConfig struct{}

func (cfg EmptyConnectorConfig) Validate() error {
	return nil
}

type ConnectorState interface{}

type EmptyConnectorState struct{}
