package wise

type Config struct {
	APIKey string `json:"apiKey" yaml:"apiKey" bson:"apiKey"`
}

func (c Config) Validate() error {
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}

	return nil
}
