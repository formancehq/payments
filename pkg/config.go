package payments

type ConnectorConfigObject interface {
	Validate() error
}

type ConnectorState interface{}
