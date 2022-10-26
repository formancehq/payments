package dummypay

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"time"
)

// Config is the configuration for the dummy payment connector.
type Config struct {
	// Directory is the directory where the files are stored.
	Directory string `json:"directory" yaml:"directory" bson:"directory"`

	// FilePollingPeriod is the period between file polling.
	FilePollingPeriod Duration `json:"filePollingPeriod" yaml:"filePollingPeriod" bson:"filePollingPeriod"`

	// FileGenerationPeriod is the period between file generation
	FileGenerationPeriod Duration `json:"fileGenerationPeriod" yaml:"fileGenerationPeriod" bson:"fileGenerationPeriod"`
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
	if cfg.FilePollingPeriod <= 0 {
		return fmt.Errorf("filePollingPeriod must be greater than 0: %w",
			ErrFilePollingPeriodInvalid)
	}

	// check if file generation period is set properly
	if cfg.FileGenerationPeriod <= 0 {
		return fmt.Errorf("fileGenerationPeriod must be greater than 0: %w",
			ErrFileGenerationPeriodInvalid)
	}

	return nil
}

type Duration time.Duration

func (d *Duration) Duration() time.Duration {
	return time.Duration(*d)
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}

	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}

		*d = Duration(tmp)

		return nil
	default:
		return errors.New("invalid duration")
	}
}
