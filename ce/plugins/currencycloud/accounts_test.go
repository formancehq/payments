package currencycloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/ce/plugins/currencycloud/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("CurrencyCloud Plugin Accounts", func() {
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
			sampleAccounts []*client.Account
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleAccounts = make([]*client.Account, 0)
			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, &client.Account{
					ID:          fmt.Sprintf("%d", i),
					AccountName: fmt.Sprintf("Account %d", i),
					CreatedAt:   now.Add(-time.Duration(50-i) * time.Minute).UTC(),
					UpdatedAt:   now.Add(-time.Duration(50-i) * time.Minute).UTC(),
				})
			}
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 1, 60).Return(
				[]*client.Account{},
				-1,
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

			m.EXPECT().GetAccounts(gomock.Any(), 1, 60).Return(
				[]*client.Account{},
				-1,
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
			Expect(state.LastCreatedAt.IsZero()).To(BeTrue())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 1, 60).Return(
				sampleAccounts,
				-1,
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
			Expect(state.LastCreatedAt).To(Equal(sampleAccounts[49].CreatedAt))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 1, 40).Return(
				sampleAccounts[:40],
				2,
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
			Expect(state.LastCreatedAt).To(Equal(sampleAccounts[39].CreatedAt))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastCreatedAt": "%s", "lastProcessedIDs": ["%s"]}`, sampleAccounts[38].CreatedAt.Format(time.RFC3339Nano), sampleAccounts[38].ID)),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 1, 40).Return(
				sampleAccounts[:40],
				2,
				nil,
			)

			m.EXPECT().GetAccounts(gomock.Any(), 2, 40).Return(
				sampleAccounts[41:],
				-1,
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastCreatedAt).To(Equal(sampleAccounts[49].CreatedAt))
		})

		It("walks a same-second group larger than PageSize across cycles without stalling", func(ctx SpecContext) {
			// Five accounts all sharing the same CreatedAt second, fetched three per
			// page so the group spans page 1 (a0,a1,a2) and a SHORT final page 2
			// (a3,a4). The monotonic LastPage cursor resumes the forward scan (no
			// full rescan from page 1 once past it) and the processed-ID set dedups
			// the same-second rows on the re-read page; a single LastProcessedID
			// would oscillate on the multi-row final page (re-emitting a3/a4 forever).
			ts := now.Add(-time.Hour).UTC()
			mk := func(id string) *client.Account {
				return &client.Account{
					ID:          id,
					AccountName: "name-" + id,
					CreatedAt:   ts,
					UpdatedAt:   ts,
				}
			}
			all := []*client.Account{mk("a0"), mk("a1"), mk("a2"), mk("a3"), mk("a4")}
			refs := func(as []models.PSPAccount) []string {
				out := make([]string, len(as))
				for i := range as {
					out[i] = as[i].Reference
				}
				return out
			}

			// Cycle 1: fresh state, page 1 -> a0, a1, a2.
			m.EXPECT().GetAccounts(gomock.Any(), 1, 3).Return(all[0:3], 2, nil)
			resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: []byte(`{}`), PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(Equal([]string{"a0", "a1", "a2"}))
			Expect(resp.HasMore).To(BeTrue())

			// Cycle 2: rescan page 1 (all skipped via the set) then page 2 -> a3, a4.
			m.EXPECT().GetAccounts(gomock.Any(), 1, 3).Return(all[0:3], 2, nil)
			m.EXPECT().GetAccounts(gomock.Any(), 2, 3).Return(all[3:5], -1, nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(Equal([]string{"a3", "a4"}))

			// Cycle 3: LastPage is now 2, so we resume there (NOT page 1) and re-read
			// only the last page. Every row is in the processed-ID set, so it returns
			// nothing (anti-oscillation) — and crucially does not rescan history.
			m.EXPECT().GetAccounts(gomock.Any(), 2, 3).Return(all[3:5], -1, nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(BeEmpty())

			// Cycle 4: a newer-second account a5 appears on the (formerly short)
			// page 2. Resuming at page 2, the set skips a3/a4 and reaches a5.
			ts2 := ts.Add(time.Second)
			a5 := &client.Account{ID: "a5", AccountName: "name-a5", CreatedAt: ts2, UpdatedAt: ts2}
			m.EXPECT().GetAccounts(gomock.Any(), 2, 3).Return([]*client.Account{all[3], all[4], a5}, -1, nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(Equal([]string{"a5"}))
		})
	})
})
