package dummypay

import (
	"context"
	"testing"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/stretchr/testify/assert"
)

// TestLoader tests the loader.
func TestLoader(t *testing.T) {
	t.Parallel()

	config := Config{}
	logger := sharedlogging.GetLogger(context.Background())

	loader := NewLoader()

	assert.Equal(t, connectorName, loader.Name())
	assert.Equal(t, 10, loader.AllowTasks())
	assert.Equal(t, Config{
		FilePollingPeriod:    10 * time.Second,
		FileGenerationPeriod: 5 * time.Second,
	}, loader.ApplyDefaults(config))

	assert.EqualValues(t, newConnector(logger, config, newFS()), loader.Load(logger, config))
}
