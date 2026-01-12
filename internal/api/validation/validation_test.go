package validation_test

import (
	"testing"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV3Handlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation Suite")
}

var _ = Describe("Validator custom type checks", func() {
	var (
		validate *validation.Validator
	)
	BeforeEach(func() {
		validate = validation.NewValidator()
	})

	Context("validation errors for various custom tags", func() {
		type CustomStruct struct {
			ConnectorID              string                       `validate:"omitempty,connectorID"`
			ConnectorIDNullable      *string                      `validate:"omitempty,connectorID"`
			AccountID                string                       `validate:"omitempty,accountID"`
			AccountIDNullable        *string                      `validate:"omitempty,accountID"`
			AccountType              string                       `validate:"omitempty,accountType"`
			AccountTypeNullable      *string                      `validate:"omitempty,accountType"`
			PaymentType              models.PaymentType           `validate:"omitempty,paymentType"`
			PaymentTypeStr           string                       `validate:"omitempty,paymentType"`
			PaymentScheme            models.PaymentScheme         `validate:"omitempty,paymentScheme"`
			PaymentSchemeStr         string                       `validate:"omitempty,paymentScheme"`
			PaymentInitiationType    models.PaymentInitiationType `validate:"omitempty,paymentInitiationType"`
			PaymentInitiationTypeStr string                       `validate:"omitempty,paymentInitiationType"`
			Asset                    string                       `validate:"omitempty,asset"`
			AssetNullable            *string                      `validate:"omitempty,asset"`
			PhoneNumber              string                       `validate:"omitempty,phoneNumber"`
			PhoneNumberNullable      *string                      `validate:"omitempty,phoneNumber"`
			Email                    string                       `validate:"omitempty,email"`
			EmailNullable            *string                      `validate:"omitempty,email"`
			Locale                   string                       `validate:"omitempty,locale"`
			LocaleNullable           *string                      `validate:"omitempty,locale"`
		}

		DescribeTable("non conforming values",
			func(tag, fieldName string, val any) {
				vErrs, err := validate.Validate(val)
				Expect(err).ToNot(BeNil())
				Expect(vErrs).To(HaveLen(1))

				fieldErr := vErrs[0]
				Expect(fieldErr.ActualTag()).To(Equal(tag))
				Expect(fieldErr.Field()).To(Equal(fieldName))
			},
			// connectorID
			Entry("connectorID: invalid value of string on required field", "connectorID", "StringFieldName", struct {
				StringFieldName string `validate:"required,connectorID"`
			}{StringFieldName: "invalid"}),
			Entry("connectorID: invalid value of string", "connectorID", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,connectorID"`
			}{StringFieldName: "invalid"}),
			Entry("connectorID: invalid value on string pointer on required field", "connectorID", "PointerFieldName", struct {
				PointerFieldName *string `validate:"required,connectorID"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("connectorID: invalid value on string pointer", "connectorID", "PointerFieldName", struct {
				PointerFieldName *string `validate:"omitempty,connectorID"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("connectorID: unsupported type for this matcher", "connectorID", "FieldName", struct {
				FieldName int `validate:"connectorID"`
			}{FieldName: 44}),

			// accountID
			Entry("accountID: invalid value of string on required field", "accountID", "StringFieldName", struct {
				StringFieldName string `validate:"required,accountID"`
			}{StringFieldName: "invalid"}),
			Entry("accountID: invalid value of string", "accountID", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,accountID"`
			}{StringFieldName: "invalid"}),
			Entry("accountID: invalid value on string pointer on required field", "accountID", "PointerFieldName", struct {
				PointerFieldName *string `validate:"required,accountID"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("accountID: invalid value on string pointer", "accountID", "PointerFieldName", struct {
				PointerFieldName *string `validate:"omitempty,accountID"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("accountID: unsupported type for this matcher", "accountID", "FieldName", struct {
				FieldName int `validate:"accountID"`
			}{FieldName: 34}),

			// accountType
			Entry("accountType: invalid value of string on required field", "accountType", "StringFieldName", struct {
				StringFieldName string `validate:"required,accountType"`
			}{StringFieldName: "invalid"}),
			Entry("accountType: invalid value of string", "accountType", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,accountType"`
			}{StringFieldName: "invalid"}),
			Entry("accountType: invalid value on string pointer on required field", "accountType", "PointerFieldName", struct {
				PointerFieldName *string `validate:"required,accountType"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("accountType: invalid value on string pointer", "accountType", "PointerFieldName", struct {
				PointerFieldName *string `validate:"omitempty,accountType"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("accountType: unsupported type for this matcher", "accountType", "FieldName", struct {
				FieldName int `validate:"accountType"`
			}{FieldName: 0}),

			// paymentType
			Entry("paymentType: invalid value of string on required field", "paymentType", "StringFieldName", struct {
				StringFieldName string `validate:"required,paymentType"`
			}{StringFieldName: "invalid"}),
			Entry("paymentType: invalid value of string", "paymentType", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,paymentType"`
			}{StringFieldName: "invalid"}),
			Entry("paymentType: invalid value on string pointer on required field", "paymentType", "PointerFieldName", struct {
				PointerFieldName *string `validate:"required,paymentType"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("paymentType: invalid value on string pointer", "paymentType", "PointerFieldName", struct {
				PointerFieldName *string `validate:"omitempty,paymentType"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("paymentType: unsupported type for this matcher", "paymentType", "FieldName", struct {
				FieldName int `validate:"paymentType"`
			}{FieldName: 34}),

			// paymentScheme
			Entry("paymentScheme: invalid value of string on required field", "paymentScheme", "StringFieldName", struct {
				StringFieldName string `validate:"required,paymentScheme"`
			}{StringFieldName: "invalid"}),
			Entry("paymentScheme: invalid value of string", "paymentScheme", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,paymentScheme"`
			}{StringFieldName: "invalid"}),
			Entry("paymentScheme: invalid value on string pointer on required field", "paymentScheme", "PointerFieldName", struct {
				PointerFieldName *string `validate:"required,paymentScheme"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("paymentScheme: invalid value on string pointer", "paymentScheme", "PointerFieldName", struct {
				PointerFieldName *string `validate:"omitempty,paymentScheme"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("paymentScheme: unsupported type for this matcher", "paymentScheme", "FieldName", struct {
				FieldName int `validate:"paymentScheme"`
			}{FieldName: 34}),

			// paymentInitiationType
			Entry("paymentInitiationType: invalid value of string on required field", "paymentInitiationType", "StringFieldName", struct {
				StringFieldName string `validate:"required,paymentInitiationType"`
			}{StringFieldName: "invalid"}),
			Entry("paymentInitiationType: invalid value of string", "paymentInitiationType", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,paymentInitiationType"`
			}{StringFieldName: "invalid"}),
			Entry("paymentInitiationType: invalid value on string pointer on required field", "paymentInitiationType", "PointerFieldName", struct {
				PointerFieldName *string `validate:"required,paymentInitiationType"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("paymentInitiationType: invalid value on string pointer", "paymentInitiationType", "PointerFieldName", struct {
				PointerFieldName *string `validate:"omitempty,paymentInitiationType"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("paymentInitiationType: unsupported type for this matcher", "paymentInitiationType", "FieldName", struct {
				FieldName int `validate:"paymentInitiationType"`
			}{FieldName: 34}),

			// asset - now accepts any UMN format (CURRENCY or CURRENCY/PRECISION)
			Entry("asset: invalid - negative precision", "asset", "StringFieldName", struct {
				StringFieldName string `validate:"required,asset"`
			}{StringFieldName: "USD/-1"}),
			Entry("asset: invalid - non-numeric precision", "asset", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,asset"`
			}{StringFieldName: "USD/abc"}),
			Entry("asset: invalid - empty currency code", "asset", "PointerFieldName", struct {
				PointerFieldName *string `validate:"required,asset"`
			}{PointerFieldName: pointer.For("/2")}),
			Entry("asset: invalid - too many slashes", "asset", "PointerFieldName", struct {
				PointerFieldName *string `validate:"omitempty,asset"`
			}{PointerFieldName: pointer.For("USD/2/3")}),
			Entry("asset: unsupported type for this matcher", "asset", "FieldName", struct {
				FieldName int `validate:"asset"`
			}{FieldName: 34}),

			// phoneNumber
			Entry("phoneNumber: invalid value of string on required field", "phoneNumber", "StringFieldName", struct {
				StringFieldName string `validate:"required,phoneNumber"`
			}{StringFieldName: "invalid"}),
			Entry("phoneNumber: invalid value of string", "phoneNumber", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,phoneNumber"`
			}{StringFieldName: "invalid"}),
			Entry("phoneNumber: invalid value of string on required field", "phoneNumber", "StringFieldName", struct {
				StringFieldName *string `validate:"required,phoneNumber"`
			}{StringFieldName: pointer.For("invalid")}),
			Entry("phoneNumber: invalid value of string", "phoneNumber", "StringFieldName", struct {
				StringFieldName *string `validate:"omitempty,phoneNumber"`
			}{StringFieldName: pointer.For("invalid")}),
			Entry("phoneNumber: unsupported type for this matcher", "phoneNumber", "FieldName", struct {
				FieldName int `validate:"phoneNumber"`
			}{FieldName: 34}),

			// email
			Entry("email: invalid value of string on required field", "email", "StringFieldName", struct {
				StringFieldName string `validate:"required,email"`
			}{StringFieldName: "invalid"}),
			Entry("email: invalid value of string", "email", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,email"`
			}{StringFieldName: "invalid"}),
			Entry("email: invalid value of string on required field", "email", "StringFieldName", struct {
				StringFieldName *string `validate:"required,email"`
			}{StringFieldName: pointer.For("invalid")}),
			Entry("email: invalid value of string", "email", "StringFieldName", struct {
				StringFieldName *string `validate:"omitempty,email"`
			}{StringFieldName: pointer.For("invalid")}),
			Entry("email: unsupported type for this matcher", "email", "FieldName", struct {
				FieldName int `validate:"email"`
			}{FieldName: 34}),

			// email
			Entry("locale: invalid value of string on required field", "locale", "StringFieldName", struct {
				StringFieldName string `validate:"required,locale"`
			}{StringFieldName: "invalid"}),
			Entry("locale: invalid value of string", "locale", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,locale"`
			}{StringFieldName: "invalid"}),
			Entry("locale: invalid value of string on required field", "locale", "StringFieldName", struct {
				StringFieldName *string `validate:"required,locale"`
			}{StringFieldName: pointer.For("invalid")}),
			Entry("locale: invalid value of string", "locale", "StringFieldName", struct {
				StringFieldName *string `validate:"omitempty,locale"`
			}{StringFieldName: pointer.For("invalid")}),
			Entry("locale: unsupported type for this matcher", "locale", "FieldName", struct {
				FieldName int `validate:"locale"`
			}{FieldName: 34}),
		)

		It("connectorID supports expected values", func(ctx SpecContext) {
			connID := models.ConnectorID{Reference: uuid.New()}
			_, err := validate.Validate(CustomStruct{
				ConnectorID:         connID.String(),
				ConnectorIDNullable: pointer.For(connID.String()),
			})
			Expect(err).To(BeNil())
		})
		It("accountID supports expected values", func(ctx SpecContext) {
			accID := models.AccountID{Reference: "ref"}
			_, err := validate.Validate(CustomStruct{
				AccountID:         accID.String(),
				AccountIDNullable: pointer.For(accID.String()),
			})
			Expect(err).To(BeNil())
		})
		It("accountType supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				AccountType:         string(models.ACCOUNT_TYPE_EXTERNAL),
				AccountTypeNullable: pointer.For(string(models.ACCOUNT_TYPE_INTERNAL)),
			})
			Expect(err).To(BeNil())
		})
		It("paymentType supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				PaymentType:    models.PAYMENT_TYPE_PAYOUT,
				PaymentTypeStr: models.PAYMENT_TYPE_TRANSFER.String(),
			})
			Expect(err).To(BeNil())
		})
		It("paymentScheme supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				PaymentScheme:    models.PAYMENT_SCHEME_CARD_ALIPAY,
				PaymentSchemeStr: models.PAYMENT_SCHEME_CARD_AMEX.String(),
			})
			Expect(err).To(BeNil())
		})
		It("paymentInitiationType supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				PaymentInitiationType:    models.PAYMENT_INITIATION_TYPE_PAYOUT,
				PaymentInitiationTypeStr: models.PAYMENT_INITIATION_TYPE_TRANSFER.String(),
			})
			Expect(err).To(BeNil())
		})
		It("asset supports expected values", func(ctx SpecContext) {
			// ISO 4217 currencies
			_, err := validate.Validate(CustomStruct{
				Asset:         "JPY/0",
				AssetNullable: pointer.For("cad/2"),
			})
			Expect(err).To(BeNil())

			// Non-ISO currencies (crypto)
			_, err = validate.Validate(CustomStruct{
				Asset:         "BTC/8",
				AssetNullable: pointer.For("ETH/18"),
			})
			Expect(err).To(BeNil())

			// Custom tokens without precision
			_, err = validate.Validate(CustomStruct{
				Asset:         "COIN",
				AssetNullable: pointer.For("TOKEN"),
			})
			Expect(err).To(BeNil())
		})
		It("phoneNumber supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				PhoneNumber:         "+330612131415",
				PhoneNumberNullable: pointer.For("+330612131415"),
			})
			Expect(err).To(BeNil())

			_, err = validate.Validate(CustomStruct{
				PhoneNumber:         "0612131415",
				PhoneNumberNullable: pointer.For("0612131415"),
			})
			Expect(err).To(BeNil())

			_, err = validate.Validate(CustomStruct{
				PhoneNumber:         "+1 (555) 555-1234",
				PhoneNumberNullable: pointer.For("+1 (555) 555-1234"),
			})
			Expect(err).To(BeNil())

			_, err = validate.Validate(CustomStruct{
				PhoneNumber:         "00 1 202 555 0123",
				PhoneNumberNullable: pointer.For("00 1 202 555 0123"),
			})
			Expect(err).To(BeNil())
		})
		It("email supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				Email:         "dev@formance.com",
				EmailNullable: pointer.For("dev@formance.com"),
			})
			Expect(err).To(BeNil())
		})
		It("language supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				Locale:         "en",
				LocaleNullable: pointer.For("en"),
			})
			Expect(err).To(BeNil())

			_, err = validate.Validate(CustomStruct{
				Locale:         "fr_FR",
				LocaleNullable: pointer.For("fr_FR"),
			})
			Expect(err).To(BeNil())

			_, err = validate.Validate(CustomStruct{
				Locale:         "iv-u",
				LocaleNullable: pointer.For("iv"),
			})
			Expect(err).ToNot(BeNil())

		})
	})
})
