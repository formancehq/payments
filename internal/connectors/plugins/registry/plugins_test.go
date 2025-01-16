package registry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Suite")
}

var _ = Describe("Register Plugin", func() {
	type Config struct {
		RequiredString   string        `json:"requiredString" validate:"required"`
		OptionalString   string        `json:"optionalString" validate:""`
		RequiredUint     uint          `json:"requiredUint" validate:"required"`
		OptionalUint     uint          `json:"optionalUint" validate:""`
		RequiredDuration time.Duration `json:"requiredDuration" validate:"required"`
		OptionalDuration time.Duration `json:"optionalDuration" validate:""`
		WithJsonMetadata string        `json:"withJsonMetadata,omitempty" validate:""`
	}
	var (
		ctrl         *gomock.Controller
		name         = "plugin-name"
		capabilities = []models.Capability{}
		conf         = Config{}
		fn           = func(_ string, _ logging.Logger, _ json.RawMessage) (models.Plugin, error) {
			plg := models.NewMockPlugin(ctrl)
			return plg, nil
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	Context("population of plugin configuration", func() {
		RegisterPlugin(name, fn, capabilities, conf)
		It("can parse a required string", func(ctx SpecContext) {
			configs := GetConfigs()
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["requiredString"]).NotTo(BeNil())
			Expect(c["requiredString"].DataType).To(Equal(TypeString))
			Expect(c["requiredString"].Required).To(BeTrue())
			Expect(c["requiredString"].DefaultValue).To(Equal(""))
		})

		It("can parse an optional string", func(ctx SpecContext) {
			configs := GetConfigs()
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["optionalString"]).NotTo(BeNil())
			Expect(c["optionalString"].DataType).To(Equal(TypeString))
			Expect(c["optionalString"].Required).To(BeFalse())
			Expect(c["optionalString"].DefaultValue).To(Equal(""))
		})

		It("can parse a required unsigned integer", func(ctx SpecContext) {
			configs := GetConfigs()
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["requiredUint"]).NotTo(BeNil())
			Expect(c["requiredUint"].DataType).To(Equal(TypeUnsignedInteger))
			Expect(c["requiredUint"].Required).To(BeTrue())
			Expect(c["requiredUint"].DefaultValue).To(Equal(""))
		})

		It("can parse an optional unsigned integer", func(ctx SpecContext) {
			configs := GetConfigs()
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["optionalUint"]).NotTo(BeNil())
			Expect(c["optionalUint"].DataType).To(Equal(TypeUnsignedInteger))
			Expect(c["optionalUint"].Required).To(BeFalse())
			Expect(c["optionalUint"].DefaultValue).To(Equal(""))
		})

		It("can parse a required duration", func(ctx SpecContext) {
			configs := GetConfigs()
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["requiredDuration"]).NotTo(BeNil())
			Expect(c["requiredDuration"].DataType).To(Equal(TypeDurationNs))
			Expect(c["requiredDuration"].Required).To(BeTrue())
			Expect(c["requiredDuration"].DefaultValue).To(Equal(""))
		})

		It("can parse an optional duration", func(ctx SpecContext) {
			configs := GetConfigs()
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["optionalDuration"]).NotTo(BeNil())
			Expect(c["optionalDuration"].DataType).To(Equal(TypeDurationNs))
			Expect(c["optionalDuration"].Required).To(BeFalse())
			Expect(c["optionalDuration"].DefaultValue).To(Equal(""))
		})

		It("can extract json field name when extra metadata present", func(ctx SpecContext) {
			configs := GetConfigs()
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["withJsonMetadata"]).NotTo(BeNil())
			Expect(c["withJsonMetadata"].DataType).To(Equal(TypeString))
			Expect(c["withJsonMetadata"].Required).To(BeFalse())
			Expect(c["withJsonMetadata"].DefaultValue).To(Equal(""))
		})
	})
})
