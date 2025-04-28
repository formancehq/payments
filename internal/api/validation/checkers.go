package validation

import (
	"regexp"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/models"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

var (
	phoneNumberRegexp = regexp.MustCompile(`^\+?\d{1,4}?[-.\s]?\(?\d{1,3}?\)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}$`)
	emailRegexp       = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

//nolint:errcheck
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

func IsAccountType(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}

	if models.AccountTypeFromString(str) == models.ACCOUNT_TYPE_UNKNOWN {
		return false
	}
	return true
}

func IsPaymentScheme(fl validator.FieldLevel) bool {
	_, ok := fl.Field().Interface().(models.PaymentScheme)
	if ok {
		return true
	}

	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	if _, err := models.PaymentSchemeFromString(str); err != nil {
		return false
	}
	return true
}

func IsPaymentStatus(fl validator.FieldLevel) bool {
	_, ok := fl.Field().Interface().(models.PaymentStatus)
	if ok {
		return true
	}

	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	if _, err := models.PaymentStatusFromString(str); err != nil {
		return false
	}
	return true
}

func IsPaymentType(fl validator.FieldLevel) bool {
	_, ok := fl.Field().Interface().(models.PaymentType)
	if ok {
		return true
	}

	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	if _, err := models.PaymentTypeFromString(str); err != nil {
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
	if err != nil { //nolint:gosimple
		return false
	}
	return true
}

func IsPhoneNumber(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}

	return phoneNumberRegexp.MatchString(str)
}

func IsEmail(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}

	return emailRegexp.MatchString(str)
}
