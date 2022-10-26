package dummypay

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestConfigString tests the string representation of the config.
func TestConfigString(t *testing.T) {
	t.Parallel()

	config := Config{
		Directory:            "test",
		FilePollingPeriod:    Duration(time.Second),
		FileGenerationPeriod: Duration(time.Minute),
	}

	assert.Equal(t, "directory: test, filePollingPeriod: 1s, fileGenerationPeriod: 1m0s", config.String())
}

// TestConfigValidate tests the validation of the config.
func TestConfigValidate(t *testing.T) {
	t.Parallel()

	var config Config

	// fail on missing directory
	assert.EqualError(t, config.Validate(), ErrMissingDirectory.Error())

	// fail on missing RW access to directory
	config.Directory = "/non-existing"
	assert.Error(t, config.Validate())

	// set directory with RW access
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}

	config.Directory = userHomeDir

	// fail on invalid file polling period
	config.FilePollingPeriod = -1
	assert.ErrorIs(t, config.Validate(), ErrFilePollingPeriodInvalid)

	// fail on invalid file generation period
	config.FilePollingPeriod = 1
	config.FileGenerationPeriod = -1
	assert.ErrorIs(t, config.Validate(), ErrFileGenerationPeriodInvalid)

	// success
	config.FileGenerationPeriod = 1
	assert.NoError(t, config.Validate())
}
