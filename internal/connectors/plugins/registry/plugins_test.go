package registry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
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
	type UnhandledType struct{}
	type Config struct {
		RequiredString   string        `json:"requiredString" validate:"required"`
		OptionalString   string        `json:"optionalString" validate:""`
		RequiredUint     uint          `json:"requiredUint" validate:"required"`
		OptionalUint     uint          `json:"optionalUint" validate:""`
		RequiredDuration time.Duration `json:"requiredDuration" validate:"required"`
		OptionalDuration time.Duration `json:"optionalDuration" validate:""`
		WithJsonMetadata string        `json:"withJsonMetadata,omitempty" validate:""`

		NilJsonTag      UnhandledType `json:"-"`
		unexportedField UnhandledType //nolint:unused
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
		RegisterPlugin(DummyPSPName, fn, capabilities, conf)
		It("can parse a required string", func(ctx SpecContext) {
			configs := GetConfigs(false)
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["requiredString"]).NotTo(BeNil())
			Expect(c["requiredString"].DataType).To(Equal(TypeString))
			Expect(c["requiredString"].Required).To(BeTrue())
			Expect(c["requiredString"].DefaultValue).To(Equal(""))
		})

		It("can parse an optional string", func(ctx SpecContext) {
			configs := GetConfigs(false)
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["optionalString"]).NotTo(BeNil())
			Expect(c["optionalString"].DataType).To(Equal(TypeString))
			Expect(c["optionalString"].Required).To(BeFalse())
			Expect(c["optionalString"].DefaultValue).To(Equal(""))
		})

		It("can parse a required unsigned integer", func(ctx SpecContext) {
			configs := GetConfigs(false)
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["requiredUint"]).NotTo(BeNil())
			Expect(c["requiredUint"].DataType).To(Equal(TypeUnsignedInteger))
			Expect(c["requiredUint"].Required).To(BeTrue())
			Expect(c["requiredUint"].DefaultValue).To(Equal(""))
		})

		It("can parse an optional unsigned integer", func(ctx SpecContext) {
			configs := GetConfigs(false)
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["optionalUint"]).NotTo(BeNil())
			Expect(c["optionalUint"].DataType).To(Equal(TypeUnsignedInteger))
			Expect(c["optionalUint"].Required).To(BeFalse())
			Expect(c["optionalUint"].DefaultValue).To(Equal(""))
		})

		It("can parse a required duration", func(ctx SpecContext) {
			configs := GetConfigs(false)
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["requiredDuration"]).NotTo(BeNil())
			Expect(c["requiredDuration"].DataType).To(Equal(TypeDurationNs))
			Expect(c["requiredDuration"].Required).To(BeTrue())
			Expect(c["requiredDuration"].DefaultValue).To(Equal(""))
		})

		It("can parse an optional duration", func(ctx SpecContext) {
			configs := GetConfigs(false)
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["optionalDuration"]).NotTo(BeNil())
			Expect(c["optionalDuration"].DataType).To(Equal(TypeDurationNs))
			Expect(c["optionalDuration"].Required).To(BeFalse())
			Expect(c["optionalDuration"].DefaultValue).To(Equal(""))
		})

		It("can extract json field name when extra metadata present", func(ctx SpecContext) {
			configs := GetConfigs(false)
			c, ok := configs[name]
			Expect(ok).To(BeTrue())
			Expect(c["withJsonMetadata"]).NotTo(BeNil())
			Expect(c["withJsonMetadata"].DataType).To(Equal(TypeString))
			Expect(c["withJsonMetadata"].Required).To(BeFalse())
			Expect(c["withJsonMetadata"].DefaultValue).To(Equal(""))
		})

		It("hides dummypay when not in debug mode", func(ctx SpecContext) {
			configs := GetConfigs(false)
			_, ok := configs[DummyPSPName]
			Expect(ok).To(BeFalse())
		})

		It("shows dummypay when in debug mode", func(ctx SpecContext) {
			configs := GetConfigs(true)
			_, ok := configs[DummyPSPName]
			Expect(ok).To(BeTrue())
		})
	})
})

var _ = Describe("Plugin Functions", func() {
	var (
		ctrl           *gomock.Controller
		logger         logging.Logger
		pluginName     = "test-plugin-functions"
		dummyPluginMock *models.MockPlugin
		originalRegistry map[string]PluginInformation
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		logger = logging.Testing()
		dummyPluginMock = models.NewMockPlugin(ctrl)
		dummyPluginMock.EXPECT().Name().Return("test-plugin-functions").AnyTimes()
		
		originalRegistry = make(map[string]PluginInformation)
		for k, v := range pluginsRegistry {
			originalRegistry[k] = v
		}
		
		capabilities := []models.Capability{models.CAPABILITY_FETCH_PAYMENTS}
		config := Config{}
		
		// Register our test plugins without clearing the registry
		RegisterPlugin(pluginName, func(_ string, _ logging.Logger, _ json.RawMessage) (models.Plugin, error) {
			return dummyPluginMock, nil
		}, capabilities, config)
	})

	AfterEach(func() {
		pluginsRegistry = make(map[string]PluginInformation)
		for k, v := range originalRegistry {
			pluginsRegistry[k] = v
		}
	})

	Context("GetPlugin", func() {
		It("returns the correct plugin for a registered provider", func(ctx SpecContext) {
			plugin, err := GetPlugin(logger, pluginName, "connector-name", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(plugin).NotTo(BeNil())
		})

		It("returns error for unknown provider", func(ctx SpecContext) {
			plugin, err := GetPlugin(logger, "unknown-plugin-name", "connector-name", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(ErrPluginNotFound.Error()))
			Expect(plugin).To(BeNil())
		})
		
		It("translates errors from plugin creation", func(ctx SpecContext) {
			errorPluginName := "error-plugin-test"
			RegisterPlugin(errorPluginName, func(_ string, _ logging.Logger, _ json.RawMessage) (models.Plugin, error) {
				return nil, models.ErrInvalidConfig
			}, []models.Capability{}, Config{})
			
			plugin, err := GetPlugin(logger, errorPluginName, "connector-name", nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(models.ErrInvalidConfig.Error())))
			Expect(plugin).To(BeNil())
		})
	})

	Context("GetCapabilities", func() {
		It("returns capabilities for a registered plugin", func(ctx SpecContext) {
			capabilities, err := GetCapabilities(pluginName)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilities).To(HaveLen(1))
			Expect(capabilities[0]).To(Equal(models.CAPABILITY_FETCH_PAYMENTS))
		})

		It("returns error for unknown plugin", func(ctx SpecContext) {
			capabilities, err := GetCapabilities("unknown-plugin-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(ErrPluginNotFound.Error()))
			Expect(capabilities).To(BeNil())
		})
	})

	Context("GetConfig", func() {
		It("returns config for a registered plugin", func(ctx SpecContext) {
			config, err := GetConfig(pluginName)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).NotTo(BeNil())
		})

		It("returns error for unknown plugin", func(ctx SpecContext) {
			config, err := GetConfig("unknown-plugin-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(ErrPluginNotFound.Error()))
			Expect(config).To(BeNil())
		})
	})
})
