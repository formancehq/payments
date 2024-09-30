package models_test

import (
	"errors"
	"testing"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestModels(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Models Suite")
}

var _ = Describe("PluginError", func() {
	var (
		err    error
		plgErr *models.PluginError
	)

	BeforeEach(func() {
		err = errors.New("some err")
		plgErr = models.NewPluginError(err)
	})
	Context("generic plugin error", func() {
		It("is marked as a retryable temporal error", func(_ SpecContext) {
			Expect(plgErr.IsRetryable).To(BeTrue())
			tmpErr := plgErr.TemporalError()
			_, ok := tmpErr.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
		})

		It("can be turned into a non retryable temporal error", func(_ SpecContext) {
			plgErr = plgErr.ForbidRetry()
			Expect(plgErr.IsRetryable).To(BeFalse())
			tmpErr := plgErr.TemporalError()
			_, ok := tmpErr.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
		})

		It("recognises httpwrapper client errors as non retryable", func(_ SpecContext) {
			plgErr = models.NewPluginError(httpwrapper.ErrStatusCodeClientError)
			Expect(plgErr.IsRetryable).To(BeFalse())
			tmpErr := plgErr.TemporalError()
			_, ok := tmpErr.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
		})
	})
})
