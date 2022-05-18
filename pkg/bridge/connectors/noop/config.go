package noop

type Config struct{}

func (c Config) Validate() error {
	return nil
}

type State struct{}
