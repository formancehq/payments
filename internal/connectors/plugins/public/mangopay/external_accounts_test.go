package mangopay

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin External Accounts", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next external accounts", func() {
		var (
			m                  *client.MockClient
			sampleBankAccounts []client.BankAccount
			now                time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			sampleBankAccounts = make([]client.BankAccount, 0)
			for i := 0; i < 50; i++ {
				sampleBankAccounts = append(sampleBankAccounts, client.BankAccount{
					ID:           fmt.Sprintf("%d", i),
					OwnerName:    fmt.Sprintf("Account %d", i),
					CreationDate: now.Add(-time.Duration(50-i) * time.Minute).UTC().Unix(),
				})
			}
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetBankAccounts(gomock.Any(), "test", 1, 60).Return(
				[]client.BankAccount{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetBankAccounts(gomock.Any(), "test", 1, 60).Return(
				[]client.BankAccount{},
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
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreationDate.IsZero()).To(BeTrue())
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetBankAccounts(gomock.Any(), "test", 1, 60).Return(
				sampleBankAccounts,
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
			createdTime := time.Unix(sampleBankAccounts[49].CreationDate, 0)
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize:    40,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetBankAccounts(gomock.Any(), "test", 1, 40).Return(
				sampleBankAccounts[:40],
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
			Expect(state.LastPage).To(Equal(1))
			createdTime := time.Unix(sampleBankAccounts[39].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			lastCreatedAt := time.Unix(sampleBankAccounts[38].CreationDate, 0)
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(fmt.Sprintf(`{"lastPage": 1, "lastCreationDate": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano))),
				PageSize:    40,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetBankAccounts(gomock.Any(), "test", 1, 40).Return(
				sampleBankAccounts[:40],
				nil,
			)

			m.EXPECT().GetBankAccounts(gomock.Any(), "test", 2, 40).Return(
				sampleBankAccounts[41:],
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
			Expect(state.LastPage).To(Equal(2))
			createdTime := time.Unix(sampleBankAccounts[49].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})
	})
})
