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

var _ = Describe("Column Plugin Accounts", func() {
	var (
		ctrl           *gomock.Controller
		mockHTTPClient *client.MockHTTPClient
		plg            models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockHTTPClient = client.NewMockHTTPClient(ctrl)
		c := client.New("test", "aseplye", "https://test.com")
		c.SetHttpClient(mockHTTPClient)
		plg = &Plugin{client: c}
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
			for i := range 50 {
				sampleAccounts = append(sampleAccounts, &client.Account{
					ID:           fmt.Sprintf("%d", i),
					Description:  fmt.Sprintf("Account %d", i),
					CurrencyCode: "USD",
					CreatedAt:    now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
				})
			}
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
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
			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get accounts: test error : "))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should return an error - invalid created_at", func(ctx SpecContext) {
			accounts := []*client.Account{{
				ID:           "acc_123",
				Description:  "Test Account",
				CurrencyCode: "USD",
				CreatedAt:    "invalid-timestamp",
				Type:         "wire",
			}}

			req := models.FetchNextAccountsRequest{
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
			).SetArg(2, client.AccountResponseWrapper[[]*client.Account]{
				BankAccounts: accounts,
			})

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(`parsing time "invalid-timestamp" as "2006-01-02T15:04:05Z07:00": cannot parse "invalid-timestamp" as "2006"`))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
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
			).SetArg(2, client.AccountResponseWrapper[[]*client.Account]{
				BankAccounts: []*client.Account{},
			})
			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
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
			).SetArg(2, client.AccountResponseWrapper[[]*client.Account]{
				BankAccounts: sampleAccounts,
			})
			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(Equal("49"))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
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
			).SetArg(2, client.AccountResponseWrapper[[]*client.Account]{
				BankAccounts: sampleAccounts[:40],
				HasMore:      true,
			})
			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())
			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastIDCreated).To(Equal("39"))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			createdAtAfter, _ := time.Parse(time.RFC3339, sampleAccounts[38].CreatedAt)
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"next_cursor": "wrY4nKh", "created_at_after": "%s"}`, createdAtAfter.UTC().Format(time.RFC3339))),
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
			).SetArg(2, client.AccountResponseWrapper[[]*client.Account]{
				BankAccounts: sampleAccounts[:40],
				HasMore:      true,
			})
			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())
			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(Equal("39"))
		})
	})
})
