package routable

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Routable fetchNextAccounts", func() {
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

	It("maps Routable settings accounts to internal PSPAccounts", func(ctx SpecContext) {
		mock.EXPECT().ListAccounts(gomock.Any(), 1, 50).Return(&client.ListAccountsResponse{
			Results: []client.Account{{
				ID:           "acc_1",
				Name:         "Operating",
				CurrencyCode: "USD",
				CreatedAt:    time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
				TypeDetails:  client.AccountTypeDetails{AvailableAmount: "1000.00"},
			}},
		}, nil)

		resp, err := plg.fetchNextAccounts(ctx, models.FetchNextAccountsRequest{PageSize: 50})
		Expect(err).To(BeNil())
		Expect(resp.Accounts).To(HaveLen(1))
		Expect(resp.Accounts[0].Reference).To(Equal("acc_1"))
		Expect(resp.Accounts[0].Name).NotTo(BeNil())
		Expect(*resp.Accounts[0].Name).To(Equal("Operating"))
		Expect(resp.Accounts[0].DefaultAsset).NotTo(BeNil())
		Expect(*resp.Accounts[0].DefaultAsset).To(ContainSubstring("USD"))
		Expect(resp.HasMore).To(BeFalse())

		var state pageState
		Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
		Expect(state.Page).To(Equal(1)) // reset because no more pages
	})

	It("advances the page cursor when more pages remain", func(ctx SpecContext) {
		mock.EXPECT().ListAccounts(gomock.Any(), 2, 25).Return(&client.ListAccountsResponse{
			Results: []client.Account{{ID: "acc_2", CreatedAt: time.Now().UTC()}},
			Links:   client.Links{Next: "/v1/settings/accounts?page=3"},
		}, nil)

		req := models.FetchNextAccountsRequest{PageSize: 25, State: json.RawMessage(`{"page":2}`)}
		resp, err := plg.fetchNextAccounts(ctx, req)
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeTrue())
		var state pageState
		Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
		Expect(state.Page).To(Equal(3))
	})

	It("propagates client errors", func(ctx SpecContext) {
		mock.EXPECT().ListAccounts(gomock.Any(), 1, 50).Return(nil, errors.New("boom"))
		_, err := plg.fetchNextAccounts(ctx, models.FetchNextAccountsRequest{PageSize: 50})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("listing accounts"))
	})
})

func TestPageStateNextPage(t *testing.T) {
	if got := (pageState{}).nextPage(); got != 1 {
		t.Errorf("zero state nextPage = %d, want 1", got)
	}
	if got := (pageState{Page: 5}).nextPage(); got != 5 {
		t.Errorf("Page=5 nextPage = %d, want 5", got)
	}
}
