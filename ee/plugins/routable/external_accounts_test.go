package routable

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/pkg/domain/plugins"
	"github.com/formancehq/payments/pkg/domain/models"
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
		Expect(resp.ExternalAccounts[0].Metadata[mappers.MetadataPrefix+"is_vendor"]).To(Equal("true"))
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

	Describe("refresh-interval throttle (24h hardcoded)", func() {
		It("stamps LastCompletedAt at end of the first walk", func(ctx SpecContext) {
			before := time.Now()
			mock.EXPECT().ListCompanies(gomock.Any(), 1, 25).Return(&client.ListCompaniesResponse{
				Results: []client.Company{{ID: "co_1", DisplayName: "Acme", CreatedAt: time.Now().UTC()}},
			}, nil)
			resp, err := plg.fetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{PageSize: 25})
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(1))

			var afterFirst pageState
			Expect(json.Unmarshal(resp.NewState, &afterFirst)).To(Succeed())
			Expect(afterFirst.Page).To(Equal(1))
			Expect(afterFirst.LastCompletedAt).To(BeTemporally(">=", before))
			Expect(afterFirst.LastCompletedAt).To(BeTemporally("<=", time.Now()))
		})

		It("skips the cycle and returns the input state unchanged when LastCompletedAt is recent", func(ctx SpecContext) {
			seed := pageState{Page: 1, LastCompletedAt: time.Now().Add(-1 * time.Hour)}
			seedJSON, _ := json.Marshal(seed)
			// No mock.EXPECT(): the throttle must skip without an HTTP call.
			resp, err := plg.fetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{PageSize: 25, State: seedJSON})
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(BeEmpty())
			Expect(resp.HasMore).To(BeFalse())
			Expect(string(resp.NewState)).To(Equal(string(seedJSON)), "skipped cycles must return the input state byte-for-byte")
		})

		It("resumes the walk once the 24h refresh interval has elapsed", func(ctx SpecContext) {
			seed := pageState{Page: 1, LastCompletedAt: time.Now().Add(-25 * time.Hour)}
			seedJSON, _ := json.Marshal(seed)
			mock.EXPECT().ListCompanies(gomock.Any(), 1, 25).Return(&client.ListCompaniesResponse{
				Results: []client.Company{{ID: "co_3", DisplayName: "Beta", CreatedAt: time.Now().UTC()}},
			}, nil)
			resp, err := plg.fetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{PageSize: 25, State: seedJSON})
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(1))
		})

		It("does not throttle mid-cycle pagination even if LastCompletedAt is recent", func(ctx SpecContext) {
			seed := pageState{Page: 5, LastCompletedAt: time.Now().Add(-1 * time.Hour)}
			seedJSON, _ := json.Marshal(seed)
			mock.EXPECT().ListCompanies(gomock.Any(), 5, 25).Return(&client.ListCompaniesResponse{
				Results: []client.Company{{ID: "co_p5", DisplayName: "Continuing", CreatedAt: time.Now().UTC()}},
			}, nil)
			resp, err := plg.fetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{PageSize: 25, State: seedJSON})
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(1))
		})
	})
})
