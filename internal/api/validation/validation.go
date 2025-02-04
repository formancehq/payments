package validation

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

func NewValidator() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())

	validate.RegisterValidation("accountID", IsAccountID, false)
	validate.RegisterValidation("connectorID", IsConnectorID, false)
	validate.RegisterValidation("paymentInitiationType", IsPaymentInitiationType, false)
	return validate
}

func fieldLevelToString(fl validator.FieldLevel) (str string, err error) {
	switch v := fl.Field().Interface().(type) {
	case *string:
		str = *fl.Field().Interface().(*string)
	case string:
		str = fl.Field().Interface().(string)
	default:
		return str, fmt.Errorf("unsupported type %T", v)
	}
	return str, nil
}

func IsConnectorID(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	if _, err := models.ConnectorIDFromString(str); err != nil {
		return false
	}
	return true
}

func IsAccountID(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	if _, err := models.AccountIDFromString(str); err != nil {
		return false
	}
	return true
}

func IsPaymentInitiationType(fl validator.FieldLevel) bool {
	_, ok := fl.Field().Interface().(models.PaymentInitiationType)
	if ok {
		return true
	}

	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	if _, err := models.PaymentInitiationTypeFromString(str); err != nil {
		return false
	}
	return true
}
