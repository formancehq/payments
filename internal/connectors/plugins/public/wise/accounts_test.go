package wise

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin Accounts", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next accounts", func() {
		var (
			sampleBalances []client.Balance
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleBalances = make([]client.Balance, 0)
			for i := 0; i < 50; i++ {
				sampleBalances = append(sampleBalances, client.Balance{
					ID:       uint64(i),
					Currency: "USD",
					Name:     "test1",
					Amount: client.BalanceAmount{
						Value:    "100",
						Currency: "USD",
					},
					CreationTime: now.Add(-time.Duration(50-i) * time.Minute).UTC(),
				})
			}
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetBalances(gomock.Any(), uint64(0)).Return(
				[]client.Balance{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetBalances(gomock.Any(), uint64(0)).Return(
				[]client.Balance{},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastAccountID).To(Equal(uint64(0)))
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetBalances(gomock.Any(), uint64(0)).Return(
				sampleBalances,
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastAccountID).To(Equal(uint64(49)))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize:    40,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetBalances(gomock.Any(), uint64(0)).Return(
				sampleBalances[:40],
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastAccountID).To(Equal(uint64(39)))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:       []byte(`{"lastAccountID": 38}`),
				PageSize:    40,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetBalances(gomock.Any(), uint64(0)).Return(
				sampleBalances,
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(11))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastAccountID).To(Equal(uint64(49)))
		})
	})
})
