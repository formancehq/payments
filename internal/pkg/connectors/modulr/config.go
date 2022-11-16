package modulr

type Config struct {
	APIKey    string `json:"apiKey" bson:"apiKey"`
	APISecret string `json:"apiSecret" bson:"apiSecret"`
	Endpoint  string `json:"endpoint" bson:"endpoint"`
}

func (c Config) Validate() error {
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}

	if c.APISecret == "" {
		return ErrMissingAPISecret
	}

	return nil
}
