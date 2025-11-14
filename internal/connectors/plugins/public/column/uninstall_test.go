package column

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/golang/mock/gomock"
)

var _ = Describe("Column Plugin Uninstall", func() {
	var (
		plg            models.Plugin
		ctrl           *gomock.Controller
		mockHTTPClient *client.MockHTTPClient
		now            time.Time
		ts             *httptest.Server
	)

	BeforeEach(func() {
		now = time.Now().UTC()
		ctrl = gomock.NewController(GinkgoT())
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"webhook_urls": []}`))
			Expect(err).To(BeNil())
		}))
		mockHTTPClient = client.NewMockHTTPClient(ctrl)
		c := client.New("test", "aseplye", ts.URL)
		c.SetHttpClient(mockHTTPClient)
		plg = &Plugin{
			client: c,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
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
				client.ListWebhookResponseWrapper[[]*client.EventSubscription]{
					WebhookEndpoints: []*client.EventSubscription{},
				},
			)

			resp, err := plg.Uninstall(ctx, models.UninstallRequest{
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
			resp, err := plg.Uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(MatchError("failed to list web hooks: list failed : "))
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})

		It("should handle webhook deletion error", func(ctx SpecContext) {
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				NewRequestMatcher("limit=100"),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(
				2,
				client.ListWebhookResponseWrapper[[]*client.EventSubscription]{
					WebhookEndpoints: []*client.EventSubscription{
						{
							ID:            "webhook-2",
							URL:           ts.URL + "/test-connector/webhook",
							CreatedAt:     now.Add(-time.Duration(5) * time.Minute).UTC().Format(time.RFC3339),
							UpdatedAt:     now.Add(-time.Duration(5) * time.Minute).UTC().Format(time.RFC3339),
							Description:   "description",
							EnabledEvents: []string{"book.transfer.completed"},
							Secret:        "secret",
							IsDisabled:    false,
						},
					},
				},
			)
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				NewRequestMatcher("limit=100&starting_after=webhook-2"),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(
				2,
				client.ListWebhookResponseWrapper[[]*client.EventSubscription]{
					WebhookEndpoints: []*client.EventSubscription{},
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

			resp, err := plg.Uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(MatchError("failed to delete web hooks: deletion failed : "))
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})

		It("should successfully uninstall and delete webhooks", func(ctx SpecContext) {
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				NewRequestMatcher("limit=100"),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ListWebhookResponseWrapper[[]*client.EventSubscription]{
				WebhookEndpoints: []*client.EventSubscription{
					{
						ID:            "webhook-1",
						URL:           "https://example.com/test-connector/webhook",
						CreatedAt:     now.Add(-time.Duration(5) * time.Minute).UTC().Format(time.RFC3339),
						UpdatedAt:     now.Add(-time.Duration(5) * time.Minute).UTC().Format(time.RFC3339),
						Description:   "description",
						EnabledEvents: []string{"book.transfer.completed"},
						Secret:        "secret",
						IsDisabled:    false,
					},
					{
						ID:            "webhook-2",
						URL:           "https://example.com/other-connector/webhook",
						CreatedAt:     now.Add(-time.Duration(5) * time.Minute).UTC().Format(time.RFC3339),
						UpdatedAt:     now.Add(-time.Duration(5) * time.Minute).UTC().Format(time.RFC3339),
						Description:   "description",
						EnabledEvents: []string{"book.transfer.completed"},
						Secret:        "secret",
						IsDisabled:    false,
					},
				},
			})
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				NewRequestMatcher("limit=100&starting_after=webhook-2"),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ListWebhookResponseWrapper[[]*client.EventSubscription]{
				WebhookEndpoints: []*client.EventSubscription{},
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

			resp, err := plg.Uninstall(ctx, models.UninstallRequest{
				ConnectorID: "test-connector",
			})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})
	})
})
