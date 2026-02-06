package increase

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/pkg/connectors/increase/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Increase Plugin Accounts", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next accounts", func() {
		var (
			mockHTTPClient *client.MockHTTPClient
			sampleAccounts []*client.Account
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com", "we5432345")
			plg.client.SetHttpClient(mockHTTPClient)
			now = time.Now().UTC()

			sampleAccounts = make([]*client.Account, 0)
			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, &client.Account{
					ID:        fmt.Sprintf("%d", i),
					Name:      fmt.Sprintf("Account %d", i),
					Currency:  "USD",
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
				})
			}
		})

		It("should return an error - invalid state", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
				State:    []byte(`{invalid json`),
				PageSize: 10,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("invalid character 'i' looking for beginning of object key string"))
			Expect(resp).To(Equal(connector.FetchNextAccountsResponse{}))
		})

		It("should return an error - invalid created_at", func() {
			accounts := []*client.Account{{
				ID:        "acc_123",
				Name:      "Test Account",
				Currency:  "USD",
				CreatedAt: "invalid-timestamp",
				Type:      "CHECKING",
				Bank:      "test_bank",
				Status:    "ACTIVE",
			}}

			resp, err := plg.fillAccounts(accounts, make([]connector.PSPAccount, 0), 10)

			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(`parsing time "invalid-timestamp" as "2006-01-02T15:04:05Z07:00": cannot parse "invalid-timestamp" as "2006"`))
			Expect(resp).To(BeNil())
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
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
			Expect(err).To(MatchError("failed to get accounts: test error : : status code: 0"))
			Expect(resp).To(Equal(connector.FetchNextAccountsResponse{}))
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
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
			).SetArg(2, client.ResponseWrapper[[]*client.Account]{
				Data: []*client.Account{},
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
			Expect(state.NextCursor).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
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
			).SetArg(2, client.ResponseWrapper[[]*client.Account]{
				Data: sampleAccounts,
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
			Expect(state.NextCursor).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
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
			).SetArg(2, client.ResponseWrapper[[]*client.Account]{
				Data:       sampleAccounts[:40],
				NextCursor: "wrY4nKh",
			})

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.NextCursor).To(Equal("wrY4nKh"))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			createdAtAfter, _ := time.Parse(time.RFC3339, sampleAccounts[38].CreatedAt)
			req := connector.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"next_cursor": "wrY4nKh", "created_at_after": "%s"}`, createdAtAfter.UTC().Format(time.RFC3339Nano))),
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
			).SetArg(2, client.ResponseWrapper[[]*client.Account]{
				Data:       sampleAccounts[:40],
				NextCursor: "qsdf",
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
			Expect(state.NextCursor).To(Equal("qsdf"))
		})
	})
})
