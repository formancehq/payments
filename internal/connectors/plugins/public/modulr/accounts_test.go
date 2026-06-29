package modulr

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Modulr Plugin Accounts", func() {
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
			sampleAccounts []client.Account
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleAccounts = make([]client.Account, 0)
			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, client.Account{
					ID:          fmt.Sprintf("%d", i),
					Name:        fmt.Sprintf("Account %d", i),
					Currency:    "USD",
					CreatedDate: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
				})
			}
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 0, 60, time.Time{}).Return(
				[]client.Account{},
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

			m.EXPECT().GetAccounts(gomock.Any(), 0, 60, time.Time{}).Return(
				[]client.Account{},
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

			m.EXPECT().GetAccounts(gomock.Any(), 0, 60, time.Time{}).Return(
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
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleAccounts[49].CreatedDate)
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 0, 40, time.Time{}).Return(
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
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleAccounts[39].CreatedDate)
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			lastCreatedAt, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleAccounts[38].CreatedDate)
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastCreatedAt": "%s", "lastProcessedIDs": ["%s"]}`, lastCreatedAt.UTC().Format(time.RFC3339Nano), sampleAccounts[38].ID)),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 0, 40, lastCreatedAt.UTC()).Return(
				sampleAccounts[:40],
				nil,
			)

			m.EXPECT().GetAccounts(gomock.Any(), 1, 40, lastCreatedAt.UTC()).Return(
				sampleAccounts[41:],
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
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleAccounts[49].CreatedDate)
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdTime.UTC()))
		})

		It("keeps distinct accounts that share the watermark timestamp (M-CON2)", func(ctx SpecContext) {
			createdDate := now.Add(-time.Hour).UTC().Format("2006-01-02T15:04:05.999-0700")
			ts, _ := time.Parse("2006-01-02T15:04:05.999-0700", createdDate)
			sameSecond := make([]client.Account, 0, 3)
			for _, id := range []string{"a", "b", "c"} {
				sameSecond = append(sameSecond, client.Account{
					ID:          id,
					Name:        "acc " + id,
					Currency:    "USD",
					CreatedDate: createdDate,
				})
			}

			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastCreatedAt": "%s", "lastProcessedIDs": ["a"]}`, ts.UTC().Format(time.RFC3339Nano))),
				PageSize: 40,
			}
			m.EXPECT().GetAccounts(gomock.Any(), 0, 40, ts.UTC()).Return(sameSecond, nil)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			// "a" was the already-processed boundary row; "b" and "c" share its
			// timestamp and must NOT be dropped.
			Expect(resp.Accounts).To(HaveLen(2))
			Expect([]string{resp.Accounts[0].Reference, resp.Accounts[1].Reference}).To(ConsistOf("b", "c"))
		})

		It("walks a same-second group larger than PageSize across cycles without stalling", func(ctx SpecContext) {
			createdDate := now.Add(-time.Hour).UTC().Format("2006-01-02T15:04:05.999-0700")
			ts, _ := time.Parse("2006-01-02T15:04:05.999-0700", createdDate)
			mk := func(id string) client.Account {
				return client.Account{ID: id, Name: "acc " + id, Currency: "USD", CreatedDate: createdDate}
			}
			// Five accounts all sharing the same CreatedDate second, fetched three
			// per page so the group spans page 0 (a0,a1,a2) and a SHORT final page 1
			// (a3,a4). Each cycle rescans from page 0 and skips the processed-ID set;
			// a single LastProcessedID would oscillate on the multi-row final page
			// (re-emitting a3/a4 forever) instead of settling and advancing.
			all := []client.Account{mk("a0"), mk("a1"), mk("a2"), mk("a3"), mk("a4")}
			refs := func(as []models.PSPAccount) []string {
				out := make([]string, len(as))
				for i := range as {
					out[i] = as[i].Reference
				}
				return out
			}

			// Cycle 1: fresh state, page 0 -> a0, a1, a2.
			m.EXPECT().GetAccounts(gomock.Any(), 0, 3, time.Time{}).Return(all[0:3], nil)
			resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: []byte(`{}`), PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(Equal([]string{"a0", "a1", "a2"}))
			Expect(resp.HasMore).To(BeTrue())

			// Cycle 2: rescan page 0 (all skipped via the set) then page 1 -> a3, a4.
			m.EXPECT().GetAccounts(gomock.Any(), 0, 3, ts.UTC()).Return(all[0:3], nil)
			m.EXPECT().GetAccounts(gomock.Any(), 1, 3, ts.UTC()).Return(all[3:5], nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(Equal([]string{"a3", "a4"}))

			// Cycle 3: group fully drained — every row is in the processed-ID set, so
			// the rescan returns nothing. A single LastProcessedID would re-emit a3 or
			// a4 here and oscillate; the set settles to empty.
			m.EXPECT().GetAccounts(gomock.Any(), 0, 3, ts.UTC()).Return(all[0:3], nil)
			m.EXPECT().GetAccounts(gomock.Any(), 1, 3, ts.UTC()).Return(all[3:5], nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(BeEmpty())

			// Cycle 4: a newer-second account a5 appears on the (formerly short) page
			// 1. The set skips a3/a4 and reaches a5 — no stranding.
			ts2 := ts.Add(time.Second)
			a5 := client.Account{ID: "a5", Name: "acc a5", Currency: "USD", CreatedDate: ts2.Format("2006-01-02T15:04:05.999-0700")}
			m.EXPECT().GetAccounts(gomock.Any(), 0, 3, ts.UTC()).Return(all[0:3], nil)
			m.EXPECT().GetAccounts(gomock.Any(), 1, 3, ts.UTC()).Return([]client.Account{all[3], all[4], a5}, nil)
			m.EXPECT().GetAccounts(gomock.Any(), 2, 3, ts.UTC()).Return([]client.Account{}, nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(Equal([]string{"a5"}))
		})
	})
})
