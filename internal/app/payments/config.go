package payments

type ConnectorConfigObject interface {
	Validate() error
	Marshal() ([]byte, error)
}

type EmptyConnectorConfig struct{}

func (cfg EmptyConnectorConfig) Validate() error {
	return nil
}

func (cfg EmptyConnectorConfig) Marshal() ([]byte, error) {
	return nil, nil
}
