package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestToPSPAccount(t *testing.T) {
	assert.Nil(t, models.ToPSPAccount(nil))
	assert.NotNil(t, models.ToPSPAccount(&models.Account{}))
}
