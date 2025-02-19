package increase

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Increase Plugin Uninstall", func() {
	var (
		plg *Plugin
		m   *client.MockClient
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			client: m,
		}
	})

	Context("uninstalling connector", func() {
		It("should handle empty webhooks list", func(ctx SpecContext) {
			m.EXPECT().ListEventSubscriptions(gomock.Any()).Return(
				[]*client.EventSubscription{},
				nil,
			)

			resp, err := plg.uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})

		It("should handle webhook list event error", func(ctx SpecContext) {
			m.EXPECT().ListEventSubscriptions(gomock.Any()).Return(
				nil,
				errors.New("list failed"),
			)

			resp, err := plg.uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(MatchError("list failed"))
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})

		It("should handle webhook deletion error", func(ctx SpecContext) {
			m.EXPECT().ListEventSubscriptions(gomock.Any()).Return(
				[]*client.EventSubscription{
					{
						ID:  "webhook-1",
						URL: "https://example.com/test-connector/webhook",
					},
				},
				nil,
			)

			m.EXPECT().UpdateEventSubscription(
				gomock.Any(),
				&client.UpdateEventSubscriptionRequest{Status: eventSubscriptionStatusDeleted},
				"webhook-1",
			).Return(nil, errors.New("deletion failed"))

			resp, err := plg.uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(MatchError("deletion failed"))
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})

		It("should successfully uninstall and delete webhooks", func(ctx SpecContext) {
			m.EXPECT().ListEventSubscriptions(gomock.Any()).Return(
				[]*client.EventSubscription{
					{
						ID:  "webhook-1",
						URL: "https://example.com/test-connector/webhook",
					},
					{
						ID:  "webhook-2",
						URL: "https://example.com/other-connector/webhook",
					},
				},
				nil,
			)

			m.EXPECT().UpdateEventSubscription(
				gomock.Any(),
				&client.UpdateEventSubscriptionRequest{Status: eventSubscriptionStatusDeleted},
				"webhook-1",
			).Return(&client.EventSubscription{}, nil)

			resp, err := plg.uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})
	})
})
