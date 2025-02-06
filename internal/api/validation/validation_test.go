package validation_test

import (
	"testing"

	"github.com/formancehq/go-libs/v2/pointer"
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
			PaymentInitiationType    models.PaymentInitiationType `validate:"omitempty,paymentInitiationType"`
			PaymentInitiationTypeStr string                       `validate:"omitempty,paymentInitiationType"`
			Asset                    string                       `validate:"omitempty,asset"`
			AssetNullable            *string                      `validate:"omitempty,asset"`
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

			// asset
			Entry("asset: invalid value of string on required field", "asset", "StringFieldName", struct {
				StringFieldName string `validate:"required,asset"`
			}{StringFieldName: "invalid"}),
			Entry("asset: invalid value of string", "asset", "StringFieldName", struct {
				StringFieldName string `validate:"omitempty,asset"`
			}{StringFieldName: "invalid"}),
			Entry("asset: invalid value on string pointer on required field", "asset", "PointerFieldName", struct {
				PointerFieldName *string `validate:"required,asset"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("asset: invalid value on string pointer", "asset", "PointerFieldName", struct {
				PointerFieldName *string `validate:"omitempty,asset"`
			}{PointerFieldName: pointer.For("invalid")}),
			Entry("asset: unsupported type for this matcher", "asset", "FieldName", struct {
				FieldName int `validate:"asset"`
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
		It("paymentInitiationType supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				PaymentInitiationType:    models.PAYMENT_INITIATION_TYPE_PAYOUT,
				PaymentInitiationTypeStr: models.PAYMENT_INITIATION_TYPE_TRANSFER.String(),
			})
			Expect(err).To(BeNil())
		})
		It("asset supports expected values", func(ctx SpecContext) {
			_, err := validate.Validate(CustomStruct{
				Asset:         "JPY/0",
				AssetNullable: pointer.For("cad/2"),
			})
			Expect(err).To(BeNil())
		})
	})
})
