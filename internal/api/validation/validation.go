package validation

import (
	"fmt"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/locales"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/validator/v10"

	ut "github.com/go-playground/universal-translator"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var (
	defaultLocaleStr = "en"
	defaultLocale    = en.New()
	supportedLocales []locales.Translator

	translator ut.Translator
)

func init() {
	supportedLocales = []locales.Translator{defaultLocale}
}

// Translator returns the locale for this deployment
// currently only one locale is expected
func Translator() ut.Translator {
	return translator
}

// TODO: could be added to go-libs later on so we don't have to think about it
func WrapError(w http.ResponseWriter, code string, rawErr error) {
	if errs, ok := rawErr.(validator.ValidationErrors); ok && len(errs) > 0 {
		err := fmt.Errorf("%s", errs[0].Translate(Translator()))
		api.BadRequest(w, code, err)
		return
	}
	// fallback
	api.BadRequest(w, code, rawErr)
}

func NewValidator() *validator.Validate {
	uni := ut.New(defaultLocale, supportedLocales...)
	// TODO: non-default locale could be configured for a stack if we have non-english clients who need it
	translator, _ = uni.GetTranslator(defaultLocaleStr)

	validate := validator.New(validator.WithRequiredStructEnabled())
	en_translations.RegisterDefaultTranslations(validate, translator)

	validate.RegisterValidation("accountID", IsAccountID, false)
	validate.RegisterValidation("connectorID", IsConnectorID, false)
	validate.RegisterValidation("paymentInitiationType", IsPaymentInitiationType, false)
	validate.RegisterValidation("asset", IsAsset, false)
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
