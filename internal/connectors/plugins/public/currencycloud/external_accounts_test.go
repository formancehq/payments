package currencycloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("CurrencyCloud Plugin External Accounts", func() {
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
			sampleBeneficiaries []*client.Beneficiary
			now                 time.Time
		)

		BeforeEach(func() {

			now = time.Now().UTC()

			sampleBeneficiaries = make([]*client.Beneficiary, 0)
			for i := 0; i < 50; i++ {
				sampleBeneficiaries = append(sampleBeneficiaries, &client.Beneficiary{
					ID:                    fmt.Sprintf("%d", i),
					BankAccountHolderName: fmt.Sprintf("Account %d", i),
					Name:                  fmt.Sprintf("Account %d", i),
					Currency:              "EUR",
					CreatedAt:             now.Add(-time.Duration(50-i) * time.Minute).UTC(),
				})
			}
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 60).Return(
				[]*client.Beneficiary{},
				-1,
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

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 60).Return(
				[]*client.Beneficiary{},
				-1,
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
			Expect(state.LastCreatedAt.IsZero()).To(BeTrue())
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 60).Return(
				sampleBeneficiaries,
				-1,
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
			Expect(state.LastCreatedAt).To(Equal(sampleBeneficiaries[49].CreatedAt))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 40).Return(
				sampleBeneficiaries[:40],
				2,
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
			Expect(state.LastCreatedAt).To(Equal(sampleBeneficiaries[39].CreatedAt))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastCreatedAt": "%s", "lastProcessedIDs": ["%s"]}`, sampleBeneficiaries[38].CreatedAt.Format(time.RFC3339Nano), sampleBeneficiaries[38].ID)),
				PageSize: 40,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 40).Return(
				sampleBeneficiaries[:40],
				2,
				nil,
			)

			m.EXPECT().GetBeneficiaries(gomock.Any(), 2, 40).Return(
				sampleBeneficiaries[41:],
				-1,
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
			Expect(state.LastCreatedAt).To(Equal(sampleBeneficiaries[49].CreatedAt))
		})

		It("walks a same-second group larger than PageSize across cycles without stalling", func(ctx SpecContext) {
			// Five beneficiaries all sharing the same CreatedAt second, fetched three
			// per page so the group spans page 1 (b0,b1,b2) and a SHORT final page 2
			// (b3,b4). Each cycle rescans from page 1 and skips the processed-ID set;
			// a single LastProcessedID would oscillate on the multi-row final page
			// (re-emitting b3/b4 forever) instead of settling and advancing.
			ts := now.Add(-time.Hour).UTC()
			mk := func(id string) *client.Beneficiary {
				return &client.Beneficiary{
					ID:                    id,
					BankAccountHolderName: "name-" + id,
					Name:                  "name-" + id,
					Currency:              "EUR",
					CreatedAt:             ts,
				}
			}
			all := []*client.Beneficiary{mk("b0"), mk("b1"), mk("b2"), mk("b3"), mk("b4")}
			refs := func(as []models.PSPAccount) []string {
				out := make([]string, len(as))
				for i := range as {
					out[i] = as[i].Reference
				}
				return out
			}

			// Cycle 1: fresh state, page 1 -> b0, b1, b2.
			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 3).Return(all[0:3], 2, nil)
			resp, err := plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: []byte(`{}`), PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"b0", "b1", "b2"}))
			Expect(resp.HasMore).To(BeTrue())

			// Cycle 2: rescan page 1 (all skipped via the set) then page 2 -> b3, b4.
			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 3).Return(all[0:3], 2, nil)
			m.EXPECT().GetBeneficiaries(gomock.Any(), 2, 3).Return(all[3:5], -1, nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"b3", "b4"}))

			// Cycle 3: group fully drained — every row is in the processed-ID set, so
			// the rescan returns nothing (anti-oscillation).
			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 3).Return(all[0:3], 2, nil)
			m.EXPECT().GetBeneficiaries(gomock.Any(), 2, 3).Return(all[3:5], -1, nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(BeEmpty())

			// Cycle 4: a newer-second beneficiary b5 appears on the (formerly short)
			// page 2. The set skips b3/b4 and reaches b5 — no stranding.
			ts2 := ts.Add(time.Second)
			b5 := &client.Beneficiary{ID: "b5", BankAccountHolderName: "name-b5", Name: "name-b5", Currency: "EUR", CreatedAt: ts2}
			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 3).Return(all[0:3], 2, nil)
			m.EXPECT().GetBeneficiaries(gomock.Any(), 2, 3).Return([]*client.Beneficiary{all[3], all[4], b5}, -1, nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 3})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"b5"}))
		})
	})
})
