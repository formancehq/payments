package validation

import (
	"fmt"

	"github.com/go-playground/locales"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/validator/v10"

	ut "github.com/go-playground/universal-translator"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var (
	defaultLocale    locales.Translator
	supportedLocales []locales.Translator
)

func init() {
	defaultLocale = en.New()
	supportedLocales = []locales.Translator{defaultLocale}
}

type Validator struct {
	internal   *validator.Validate
	translator ut.Translator
}

func NewValidator() *Validator {
	uni := ut.New(defaultLocale, supportedLocales...)
	// TODO: non-default locale could be configured for a stack if we have non-english clients who need it
	translator, _ := uni.GetTranslator(defaultLocale.Locale())

	validate := validator.New(validator.WithRequiredStructEnabled())
	en_translations.RegisterDefaultTranslations(validate, translator) //nolint:errcheck

	registerCustomChecker("accountID", IsAccountID, "", validate, translator)
	registerCustomChecker("accountType", IsAccountType, "", validate, translator)
	registerCustomChecker("connectorID", IsConnectorID, "", validate, translator)
	registerCustomChecker("paymentType", IsPaymentType, "", validate, translator)
	registerCustomChecker("paymentScheme", IsPaymentScheme, "", validate, translator)
	registerCustomChecker("paymentStatus", IsPaymentStatus, "", validate, translator)
	registerCustomChecker("paymentInitiationType", IsPaymentInitiationType, "", validate, translator)
	registerCustomChecker("asset", IsAsset, "", validate, translator)
	registerCustomChecker("phoneNumber", IsPhoneNumber, "", validate, translator)
	registerCustomChecker("email", IsEmail, "", validate, translator)
	registerCustomChecker("locale", IsLocale, "", validate, translator)
	return &Validator{
		internal:   validate,
		translator: translator,
	}
}

func (v *Validator) Validate(payload any) (fieldErrors validator.ValidationErrors, err error) {
	rawErr := v.internal.Struct(payload)
	if rawErr == nil {
		return fieldErrors, nil
	}

	fieldErrors, ok := rawErr.(validator.ValidationErrors)
	if !ok {
		return fieldErrors, rawErr
	}

	if len(fieldErrors) > 0 {
		return fieldErrors, fmt.Errorf("%s", fieldErrors[0].Translate(v.translator))
	}
	return fieldErrors, rawErr
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
