package dummypay

type Config struct {
	Directory string
}

func (cfg Config) Validate() error {
	if cfg.Directory == "" {
		return ErrMissingDirectory
	}

	return nil
}
