package bankingcircle

type Config struct {
	Username              string `json:"username" yaml:"username" bson:"username"`
	Password              string `json:"password" yaml:"password" bson:"password"`
	Endpoint              string `json:"endpoint" yaml:"endpoint" bson:"endpoint"`
	AuthorizationEndpoint string `json:"authorizationEndpoint" yaml:"authorizationEndpoint" bson:"authorizationEndpoint"`
}

func (c Config) Validate() error {
	if c.Username == "" {
		return ErrMissingUsername
	}

	if c.Password == "" {
		return ErrMissingPassword
	}

	if c.Endpoint == "" {
		return ErrMissingEndpoint
	}

	if c.AuthorizationEndpoint == "" {
		return ErrMissingAuthorizationEndpoint
	}

	return nil
}
