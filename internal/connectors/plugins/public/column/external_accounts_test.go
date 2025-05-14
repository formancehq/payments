package column

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Column Plugin External Accounts", func() {
	var (
		mockHTTPClient *client.MockHTTPClient
		plg            models.Plugin
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		mockHTTPClient = client.NewMockHTTPClient(ctrl)
		c := client.New("test", "aseplye", "https://test.com")
		c.SetHttpClient(mockHTTPClient)
		plg = &Plugin{client: c}
	})

	Context("fetching next external accounts", func() {
		var (
			sampleCounterparties []*client.Counterparties
			now                  time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()
			sampleCounterparties = make([]*client.Counterparties, 0)
			for i := range 50 {
				sampleCounterparties = append(sampleCounterparties, &client.Counterparties{
					ID:            fmt.Sprintf("%d", i),
					Description:   fmt.Sprintf("Account %d", i),
					AccountNumber: fmt.Sprintf("123454%d", i),
					CreatedAt:     now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
				})
			}
		})

		It("should return an error - get external accounts error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)
			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get external accounts: test error : "))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CounterpartiesResponseWrapper[[]*client.Counterparties]{
				Counterparties: []*client.Counterparties{},
			})
			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(BeEmpty())
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CounterpartiesResponseWrapper[[]*client.Counterparties]{
				Counterparties: sampleCounterparties,
			})
			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(Equal("49"))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CounterpartiesResponseWrapper[[]*client.Counterparties]{
				Counterparties: sampleCounterparties[:40],
				HasMore:        true,
			})
			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())
			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastIDCreated).To(Equal("39"))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"next_cursor": "%s"}`, "qwerty")),
				PageSize: 40,
			}
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CounterpartiesResponseWrapper[[]*client.Counterparties]{
				Counterparties: sampleCounterparties[:40],
				HasMore:        true,
			})
			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())
			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(Equal("39"))
		})
	})
})
