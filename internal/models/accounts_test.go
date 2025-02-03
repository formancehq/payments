package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToPSPAccount(t *testing.T) {
	assert.Nil(t, ToPSPAccount(nil))
	assert.NotNil(t, ToPSPAccount(&Account{}))
}
