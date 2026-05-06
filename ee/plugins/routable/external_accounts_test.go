package routable

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Routable fetchNextExternalAccounts", func() {
	var (
		ctrl   *gomock.Controller
		mock   *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mock = client.NewMockClient(ctrl)
		plg = &Plugin{Plugin: plugins.NewBasePlugin(), name: "routable", logger: logger, client: mock}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("maps Routable companies to external PSPAccounts without N+1 fan-out", func(ctx SpecContext) {
		mock.EXPECT().ListCompanies(gomock.Any(), 1, 50).Return(&client.ListCompaniesResponse{
			Results: []client.Company{{
				ID:           "co_1",
				DisplayName:  "Acme Inc",
				BusinessName: "Acme Incorporated",
				CreatedAt:    time.Now().UTC(),
				IsVendor:     true,
				CountryCode:  "US",
			}},
		}, nil)

		resp, err := plg.fetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{PageSize: 50})
		Expect(err).To(BeNil())
		Expect(resp.ExternalAccounts).To(HaveLen(1))
		Expect(resp.ExternalAccounts[0].Reference).To(Equal("co_1"))
		Expect(*resp.ExternalAccounts[0].Name).To(Equal("Acme Inc"))
		Expect(resp.ExternalAccounts[0].Metadata[MetadataPrefix+"is_vendor"]).To(Equal("true"))
		Expect(resp.HasMore).To(BeFalse())

		var state pageState
		Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
		Expect(state.Page).To(Equal(1))
	})

	It("falls back to business_name when display_name is empty", func(ctx SpecContext) {
		mock.EXPECT().ListCompanies(gomock.Any(), 1, 25).Return(&client.ListCompaniesResponse{
			Results: []client.Company{{ID: "co_2", BusinessName: "Inc.", CreatedAt: time.Now().UTC()}},
		}, nil)
		resp, err := plg.fetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{PageSize: 25})
		Expect(err).To(BeNil())
		Expect(*resp.ExternalAccounts[0].Name).To(Equal("Inc."))
	})
})
