package generic

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/ce/plugins/generic/client"
	genericclient "github.com/formancehq/payments/ce/plugins/generic/client/generated"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Generic Plugin Accounts", func() {
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
			sampleAccounts []genericclient.Account
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleAccounts = make([]genericclient.Account, 0)
			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, genericclient.Account{
					Id:          fmt.Sprint(i),
					AccountName: fmt.Sprintf("account-%d", i),
					CreatedAt:   now.Add(-time.Duration(50-i) * time.Minute).UTC(),
					Metadata: map[string]string{
						"foo": "bar",
					},
				})
			}
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().ListAccounts(gomock.Any(), int64(1), int64(60), time.Time{}).Return(
				[]genericclient.Account{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().ListAccounts(gomock.Any(), int64(1), int64(60), time.Time{}).Return(
				[]genericclient.Account{},
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
			Expect(state.LastCreatedAtFrom.IsZero()).To(BeTrue())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().ListAccounts(gomock.Any(), int64(1), int64(60), time.Time{}).Return(
				sampleAccounts,
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
			Expect(state.LastCreatedAtFrom.UTC()).To(Equal(sampleAccounts[49].CreatedAt.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().ListAccounts(gomock.Any(), int64(1), int64(40), time.Time{}).Return(
				sampleAccounts[:40],
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
			Expect(state.LastCreatedAtFrom.UTC()).To(Equal(sampleAccounts[39].CreatedAt.UTC()))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastCreatedAtFrom": "%s", "lastProcessedID": "%s"}`, sampleAccounts[38].CreatedAt.Format(time.RFC3339Nano), sampleAccounts[38].Id)),
				PageSize: 40,
			}

			m.EXPECT().ListAccounts(gomock.Any(), int64(1), int64(40), sampleAccounts[38].CreatedAt.UTC()).Return(
				sampleAccounts[:40],
				nil,
			)

			m.EXPECT().ListAccounts(gomock.Any(), int64(2), int64(40), sampleAccounts[38].CreatedAt.UTC()).Return(
				sampleAccounts[40:],
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
			Expect(state.LastCreatedAtFrom.UTC()).To(Equal(sampleAccounts[49].CreatedAt.UTC()))
		})

		It("walks a same-second group larger than PageSize across cycles without stalling", func(ctx SpecContext) {
			ts := now.Add(-time.Hour).UTC()
			mk := func(id string) genericclient.Account {
				return genericclient.Account{Id: id, AccountName: "name-" + id, CreatedAt: ts}
			}
			all := []genericclient.Account{mk("a0"), mk("a1"), mk("a2"), mk("a3"), mk("a4")}
			refs := func(as []models.PSPAccount) []string {
				out := make([]string, len(as))
				for i := range as {
					out[i] = as[i].Reference
				}
				return out
			}

			// Cycle 1: fresh state, page 1 -> a0, a1.
			m.EXPECT().ListAccounts(gomock.Any(), int64(1), int64(2), time.Time{}).Return(all[0:2], nil)
			resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: []byte(`{}`), PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(Equal([]string{"a0", "a1"}))
			Expect(resp.HasMore).To(BeTrue())

			// Cycle 2: page 1 re-fetched (a1 deduped) then page 2 -> a2, a3.
			m.EXPECT().ListAccounts(gomock.Any(), int64(1), int64(2), ts).Return(all[0:2], nil)
			m.EXPECT().ListAccounts(gomock.Any(), int64(2), int64(2), ts).Return(all[2:4], nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(ContainElements("a2", "a3"))
			Expect(refs(resp.Accounts)).ToNot(ContainElement("a1"))

			// Cycle 3: page 3 -> a4 (group fully drained, no stall).
			m.EXPECT().ListAccounts(gomock.Any(), int64(3), int64(2), ts).Return(all[4:5], nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(ContainElement("a4"))
		})
	})
})
