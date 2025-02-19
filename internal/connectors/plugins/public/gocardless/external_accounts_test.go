package gocardless

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Gocardless Plugin fetch next external accounts", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("when there are no external accounts", func() {
		var (
			m                  *client.MockClient
			sampleBankAccounts []client.GocardlessGenericAccount
			now                time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			sampleBankAccounts = make([]client.GocardlessGenericAccount, 0)
			for i := 0; i < 50; i++ {
				name := fmt.Sprintf("Account %d", i)
				sampleBankAccounts = append(sampleBankAccounts, client.GocardlessGenericAccount{
					ID:                fmt.Sprintf("BA%d", i),
					AccountHolderName: name,
					CreatedAt:         now.Add(-time.Duration(50-i) * time.Minute).Unix(),
					Metadata:          map[string]interface{}{"type": "external_account"},
					Currency:          "USD",
					AccountType:       "savings",
				})
			}
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
				PageSize:    60,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", 60, "", "").
				Return(sampleBankAccounts, client.Cursor{}, errors.New("get beneficiaries error"))

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("get beneficiaries error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
				PageSize:    60,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", req.PageSize, "", "").
				Return(
					[]client.GocardlessGenericAccount{},
					client.Cursor{},
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
			Expect(state.After).To(Equal(""))
			Expect(state.Before).To(Equal(""))
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
				PageSize:    60,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", req.PageSize, "", "").
				Return(sampleBankAccounts, client.Cursor{
					After:  "BA50",
					Before: "BA1",
				}, nil)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal("BA50"))
			Expect(state.Before).To(Equal("BA1"))

		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
				PageSize:    40,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", req.PageSize, "", "").
				Return(sampleBankAccounts[:40], client.Cursor{
					After:  "BA40",
					Before: "BA1",
				}, nil)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal("BA40"))
			Expect(state.Before).To(Equal("BA1"))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			lastCreationDate := time.Unix(sampleBankAccounts[38].CreatedAt, 0)

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
				PageSize:    40,
				State:       []byte(`{"after": "BA1", "before": "BA40", "lastCreationDate": "` + lastCreationDate.Format(time.RFC3339Nano) + `"}`),
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", req.PageSize, "BA1", "BA40").
				Return(sampleBankAccounts[:40], client.Cursor{
					After:  "BA40",
					Before: "BA1",
				}, nil)

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", req.PageSize, "BA40", "BA1").
				Return(sampleBankAccounts[41:], client.Cursor{
					After:  "BA50",
					Before: "BA41",
				}, nil)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal("BA50"))
			Expect(state.Before).To(Equal("BA41"))

		})

	})
})
