package validation

import (
	"regexp"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"golang.org/x/text/language"
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

func IsTradeStatus(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	status := models.TradeStatus(str)
	return status.IsValid()
}

func IsTradeSide(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	side := models.TradeSide(str)
	return side.IsValid()
}

func IsTradeInstrumentType(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	instrumentType := models.TradeInstrumentType(str)
	return instrumentType.IsValid()
}

func IsTradeExecutionModel(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	model := models.TradeExecutionModel(str)
	return model.IsValid()
}

func IsTradeOrderType(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	orderType := models.TradeOrderType(str)
	return orderType.IsValid()
}

func IsTradeTimeInForce(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	tif := models.TradeTimeInForce(str)
	return tif.IsValid()
}

func IsTradeLiquidity(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	liquidity := models.TradeLiquidity(str)
	return liquidity.IsValid()
}

func IsTradeFeeKind(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	kind := models.TradeFeeKind(str)
	return kind.IsValid()
}

func IsTradeFeeAppliedOn(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	appliedOn := models.TradeFeeAppliedOn(str)
	return appliedOn.IsValid()
}

func IsTradeLegRole(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	role := models.TradeLegRole(str)
	return role.IsValid()
}

func IsTradeLegDirection(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	dir := models.TradeLegDirection(str)
	return dir.IsValid()
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

func IsLocale(fl validator.FieldLevel) bool {
	str, err := fieldLevelToString(fl)
	if err != nil {
		return false
	}
	_, err = language.Parse(str)
	return err == nil
}
