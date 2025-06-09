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
		plg            *Plugin
		mockHTTPClient *client.MockHTTPClient
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		mockHTTPClient = client.NewMockHTTPClient(ctrl)
		plg = &Plugin{
			client: client.New("test", "aseplye", "https://test.com", "we5432345"),
		}
		plg.client.SetHttpClient(mockHTTPClient)
	})

	Context("uninstalling connector", func() {
		It("should handle empty webhooks list", func(ctx SpecContext) {
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(
				2,
				client.ResponseWrapper[[]*client.EventSubscription]{
					Data:       []*client.EventSubscription{},
					NextCursor: "qwerty",
				},
			)

			resp, err := plg.uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})

		It("should handle webhook list event error", func(ctx SpecContext) {
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("list failed"),
			)

			resp, err := plg.uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(MatchError("failed to list web hooks: list failed : : status code: 0"))
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})

		It("should handle webhook deletion error", func(ctx SpecContext) {
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(
				2,
				client.ResponseWrapper[[]*client.EventSubscription]{
					Data: []*client.EventSubscription{
						{
							ID:  "webhook-1",
							URL: "https://example.com/test-connector/webhook",
						},
					},
				},
			)

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				errors.New("deletion failed"),
			)

			resp, err := plg.uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(MatchError("failed to update web hooks: deletion failed : : status code: 0"))
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})

		It("should successfully uninstall and delete webhooks", func(ctx SpecContext) {
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.EventSubscription]{
				Data: []*client.EventSubscription{
					{
						ID:  "webhook-1",
						URL: "https://example.com/test-connector/webhook",
					},
					{
						ID:  "webhook-2",
						URL: "https://example.com/other-connector/webhook",
					},
				},
			})

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			)

			resp, err := plg.uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})
	})
})
