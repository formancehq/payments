package modulr

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/ce/plugins/modulr/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Modulr Plugin External Accounts", func() {
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

	Context("fetching next external accounts", func() {
		var (
			sampleBeneficiaries []client.Beneficiary
			now                 time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleBeneficiaries = make([]client.Beneficiary, 0)
			for i := 0; i < 50; i++ {
				sampleBeneficiaries = append(sampleBeneficiaries, client.Beneficiary{
					ID:      fmt.Sprintf("%d", i),
					Name:    fmt.Sprintf("Account %d", i),
					Created: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
				})
			}
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 0, 60, time.Time{}).Return(
				[]client.Beneficiary{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 0, 60, time.Time{}).Return(
				[]client.Beneficiary{},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastModifiedSince.IsZero()).To(BeTrue())
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 0, 60, time.Time{}).Return(
				sampleBeneficiaries,
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleBeneficiaries[49].Created)
			Expect(state.LastModifiedSince.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 0, 40, time.Time{}).Return(
				sampleBeneficiaries[:40],
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleBeneficiaries[39].Created)
			Expect(state.LastModifiedSince.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			lastCreatedAt, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleBeneficiaries[38].Created)
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastModifiedSince": "%s", "lastProcessedID": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano), sampleBeneficiaries[38].ID)),
				PageSize: 40,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 0, 40, lastCreatedAt.UTC()).Return(
				sampleBeneficiaries[:40],
				nil,
			)

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 40, lastCreatedAt.UTC()).Return(
				sampleBeneficiaries[41:],
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleBeneficiaries[49].Created)
			Expect(state.LastModifiedSince.UTC()).To(Equal(createdTime.UTC()))
		})

		It("walks a same-second group larger than PageSize across cycles without stalling", func(ctx SpecContext) {
			createdDate := now.Add(-time.Hour).UTC().Format("2006-01-02T15:04:05.999-0700")
			ts, _ := time.Parse("2006-01-02T15:04:05.999-0700", createdDate)
			mk := func(id string) client.Beneficiary {
				return client.Beneficiary{ID: id, Name: "ben " + id, Created: createdDate}
			}
			all := []client.Beneficiary{mk("b0"), mk("b1"), mk("b2"), mk("b3"), mk("b4")}
			refs := func(as []models.PSPAccount) []string {
				out := make([]string, len(as))
				for i := range as {
					out[i] = as[i].Reference
				}
				return out
			}

			// Cycle 1: page 0 -> b0, b1.
			m.EXPECT().GetBeneficiaries(gomock.Any(), 0, 2, time.Time{}).Return(all[0:2], nil)
			resp, err := plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: []byte(`{}`), PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"b0", "b1"}))

			// Cycle 2: page 0 re-fetched (b1 deduped) then page 1 -> b2, b3.
			m.EXPECT().GetBeneficiaries(gomock.Any(), 0, 2, ts.UTC()).Return(all[0:2], nil)
			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 2, ts.UTC()).Return(all[2:4], nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			// Boundary b1 is deduped; b0 (a same-second sibling on the re-fetched
			// page 0) is re-emitted by design — storage upserts dedup it. The exact
			// assertion catches any unintended extra re-emission.
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"b0", "b2", "b3"}))

			// Cycle 3: page 2 -> b4 (group fully drained on a short final page).
			m.EXPECT().GetBeneficiaries(gomock.Any(), 2, 2, ts.UTC()).Return(all[4:5], nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"b4"}))

			// Cycle 4: a newer-second beneficiary b5 lands on the short last page
			// (2). The cursor must stay on page 2 rather than advance to page 3, or
			// b5 would be stranded forever behind an empty page.
			ts2 := ts.Add(time.Second)
			b5 := client.Beneficiary{ID: "b5", Name: "ben b5", Created: ts2.Format("2006-01-02T15:04:05.999-0700")}
			m.EXPECT().GetBeneficiaries(gomock.Any(), 2, 2, ts.UTC()).Return([]client.Beneficiary{all[4], b5}, nil)
			m.EXPECT().GetBeneficiaries(gomock.Any(), 3, 2, ts.UTC()).Return([]client.Beneficiary{}, nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"b5"}))
		})
	})
})
