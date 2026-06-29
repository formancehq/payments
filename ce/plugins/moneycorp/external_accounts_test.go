package moneycorp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/ce/plugins/moneycorp/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp *Plugin ExternalAccounts", func() {
	var (
		plg models.Plugin
	)

	Context("fetch next ExternalAccounts", func() {
		var (
			ctrl *gomock.Controller
			m    *client.MockClient

			sampleExternalAccounts []*client.Recipient
			accRef                 string
			now                    time.Time
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
			accRef = "baseAcc"
			now = time.Now().UTC()

			sampleExternalAccounts = make([]*client.Recipient, 0)
			for i := 0; i < 50; i++ {
				sampleExternalAccounts = append(sampleExternalAccounts, &client.Recipient{
					ID: fmt.Sprintf("recipient-%d", i),
					Attributes: client.RecipientAttributes{
						BankAccountCurrency: "JPY",
						CreatedAt:           strings.TrimSuffix(now.Add(-time.Duration(60-i)*time.Minute).UTC().Format(time.RFC3339Nano), "Z"),
						BankAccountName:     "jpy account",
					},
				})
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - missing from payload", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing from payload in request"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 60).Return(
				[]*client.Recipient{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 60).Return(
				[]*client.Recipient{},
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
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 60).Return(
				sampleExternalAccounts,
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
			createdAtTime, _ := time.Parse(time.RFC3339Nano, sampleExternalAccounts[49].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(`{}`),
				PageSize:    40,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 40).Return(
				sampleExternalAccounts[:40],
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
			createdAtTime, _ := time.Parse(time.RFC3339Nano, sampleExternalAccounts[39].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			lastCreaatedAt, _ := time.Parse(time.RFC3339Nano, sampleExternalAccounts[38].Attributes.CreatedAt+"Z")
			req := models.FetchNextExternalAccountsRequest{
				State: []byte(fmt.Sprintf(
					`{"LastCreatedAt": "%s", "lastProcessedIDs": ["%s"]}`,
					lastCreaatedAt.Format(time.RFC3339Nano),
					sampleExternalAccounts[38].ID,
				)),
				PageSize:    40,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 40).Return(
				sampleExternalAccounts[:40],
				nil,
			)

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 1, 40).Return(
				sampleExternalAccounts[41:],
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
			createdAtTime, _ := time.Parse(time.RFC3339Nano, sampleExternalAccounts[49].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})

		It("walks a same-second group larger than PageSize across cycles without stalling", func(ctx SpecContext) {
			const createdAtStr = "2024-02-02T00:00:00"
			mk := func(id string) *client.Recipient {
				return &client.Recipient{
					ID: id,
					Attributes: client.RecipientAttributes{
						BankAccountCurrency: "JPY",
						CreatedAt:           createdAtStr,
						BankAccountName:     "jpy account",
					},
				}
			}
			// Five recipients sharing the same second, fetched three per page so the
			// group spans page 0 (m0,m1,m2) and a SHORT final page 1 (m3,m4). Each
			// cycle rescans from page 0 and skips the processed-ID set.
			all := []*client.Recipient{mk("m0"), mk("m1"), mk("m2"), mk("m3"), mk("m4")}
			refs := func(as []models.PSPAccount) []string {
				out := make([]string, len(as))
				for i := range as {
					out[i] = as[i].Reference
				}
				return out
			}
			accRef := "baseAcc"
			fromPayload := json.RawMessage(`{"reference": "baseAcc"}`)

			// Cycle 1: fresh state, page 0 -> m0, m1, m2.
			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 3).Return(all[0:3], nil)
			resp, err := plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: []byte(`{}`), PageSize: 3, FromPayload: fromPayload})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"m0", "m1", "m2"}))
			Expect(resp.HasMore).To(BeTrue())

			// Cycle 2: rescan page 0 (all skipped via the set) then page 1 -> m3, m4.
			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 3).Return(all[0:3], nil)
			m.EXPECT().GetRecipients(gomock.Any(), accRef, 1, 3).Return(all[3:5], nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 3, FromPayload: fromPayload})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"m3", "m4"}))

			// Cycle 3: group fully drained — every row is in the processed-ID set, so
			// the rescan returns nothing (anti-oscillation).
			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 3).Return(all[0:3], nil)
			m.EXPECT().GetRecipients(gomock.Any(), accRef, 1, 3).Return(all[3:5], nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 3, FromPayload: fromPayload})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(BeEmpty())

			// Cycle 4: a newer-second recipient m5 appears on the (formerly short)
			// page 1. The set skips m3/m4 and reaches m5 — no stranding.
			m5 := &client.Recipient{
				ID: "m5",
				Attributes: client.RecipientAttributes{
					BankAccountCurrency: "JPY",
					CreatedAt:           "2024-02-02T00:00:01",
					BankAccountName:     "jpy account",
				},
			}
			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 3).Return(all[0:3], nil)
			m.EXPECT().GetRecipients(gomock.Any(), accRef, 1, 3).Return([]*client.Recipient{all[3], all[4], m5}, nil)
			m.EXPECT().GetRecipients(gomock.Any(), accRef, 2, 3).Return([]*client.Recipient{}, nil)
			resp, err = plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{State: resp.NewState, PageSize: 3, FromPayload: fromPayload})
			Expect(err).To(BeNil())
			Expect(refs(resp.ExternalAccounts)).To(Equal([]string{"m5"}))
		})
	})
})
