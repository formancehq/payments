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
				State:    []byte(fmt.Sprintf(`{"lastCreatedAt": "%s", "lastProcessedID": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano), sampleAccounts[38].ID)),
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
				State:    []byte(fmt.Sprintf(`{"lastCreatedAt": "%s", "lastProcessedID": "a"}`, ts.UTC().Format(time.RFC3339Nano))),
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
			all := []client.Account{mk("a0"), mk("a1"), mk("a2"), mk("a3"), mk("a4")}
			refs := func(as []models.PSPAccount) []string {
				out := make([]string, len(as))
				for i := range as {
					out[i] = as[i].Reference
				}
				return out
			}

			// Cycle 1: page 0 -> a0, a1.
			m.EXPECT().GetAccounts(gomock.Any(), 0, 2, time.Time{}).Return(all[0:2], nil)
			resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: []byte(`{}`), PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(Equal([]string{"a0", "a1"}))

			// Cycle 2: page 0 re-fetched (a1 deduped) then page 1 -> a2, a3.
			m.EXPECT().GetAccounts(gomock.Any(), 0, 2, ts.UTC()).Return(all[0:2], nil)
			m.EXPECT().GetAccounts(gomock.Any(), 1, 2, ts.UTC()).Return(all[2:4], nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(ContainElements("a2", "a3"))
			Expect(refs(resp.Accounts)).ToNot(ContainElement("a1"))

			// Cycle 3: page 2 -> a4 (group fully drained, no stall).
			m.EXPECT().GetAccounts(gomock.Any(), 2, 2, ts.UTC()).Return(all[4:5], nil)
			resp, err = plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Accounts)).To(ContainElement("a4"))
		})
	})
})
