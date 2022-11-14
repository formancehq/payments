package dummypay

import (
	"fmt"

	"github.com/numary/payments/internal/pkg/connectors"
)

// Config is the configuration for the dummy payment connector.
type Config struct {
	// Directory is the directory where the files are stored.
	Directory string `json:"directory" yaml:"directory" bson:"directory"`

	// FilePollingPeriod is the period between file polling.
	FilePollingPeriod connectors.Duration `json:"filePollingPeriod" yaml:"filePollingPeriod" bson:"filePollingPeriod"`

	// FileGenerationPeriod is the period between file generation
	FileGenerationPeriod connectors.Duration `json:"fileGenerationPeriod" yaml:"fileGenerationPeriod" bson:"fileGenerationPeriod"`
}

// String returns a string representation of the configuration.
func (cfg Config) String() string {
	return fmt.Sprintf("directory: %s, filePollingPeriod: %s, fileGenerationPeriod: %s",
		cfg.Directory, cfg.FilePollingPeriod, cfg.FileGenerationPeriod)
}

// Validate validates the configuration.
func (cfg Config) Validate() error {
	// require directory path to be present
	if cfg.Directory == "" {
		return ErrMissingDirectory
	}

	// check if file polling period is set properly
	if cfg.FilePollingPeriod.Duration <= 0 {
		return fmt.Errorf("filePollingPeriod must be greater than 0: %w",
			ErrFilePollingPeriodInvalid)
	}

	// check if file generation period is set properly
	if cfg.FileGenerationPeriod.Duration <= 0 {
		return fmt.Errorf("fileGenerationPeriod must be greater than 0: %w",
			ErrFileGenerationPeriodInvalid)
	}

	return nil
}
