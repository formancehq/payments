package validation

import (
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/models"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

func registerCustomChecker(
	tagName string,
	fn func(validator.FieldLevel) bool,
	localeStr string,
	validate *validator.Validate,
	translator ut.Translator,
) {
	if localeStr == "" {
		localeStr = "{0} is invalid"
	}

	validate.RegisterValidation(tagName, fn, false)
	validate.RegisterTranslation(tagName, translator, func(u ut.Translator) error {
		return u.Add(tagName, localeStr, true)
	}, func(u ut.Translator, fe validator.FieldError) string {
		t, _ := u.T(tagName, fe.Field())
		return t
	})
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

func IsAsset(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}

	_, _, err = currency.GetCurrencyAndPrecisionFromAsset(currency.ISO4217Currencies, str)
	if err != nil {
		return false
	}
	return true
}
