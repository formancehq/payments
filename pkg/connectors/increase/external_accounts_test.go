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

var _ = Describe("Increase Plugin External Accounts", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next external accounts", func() {
		var (
			mockHTTPClient         *client.MockHTTPClient
			sampleExternalAccounts []*client.ExternalAccount
			now                    time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com", "we5432345")
			plg.client.SetHttpClient(mockHTTPClient)
			now = time.Now().UTC()

			sampleExternalAccounts = make([]*client.ExternalAccount, 0)
			for i := 0; i < 50; i++ {
				sampleExternalAccounts = append(sampleExternalAccounts, &client.ExternalAccount{
					ID:            fmt.Sprintf("%d", i),
					Description:   fmt.Sprintf("Account %d", i),
					AccountNumber: fmt.Sprintf("123454%d", i),
					CreatedAt:     now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
				})
			}
		})

		It("should return an error - get external accounts error", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
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
			Expect(err).To(MatchError("failed to get external accounts: test error : : status code: 0"))
			Expect(resp).To(Equal(connector.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
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
			).SetArg(2, client.ResponseWrapper[[]*client.ExternalAccount]{
				Data: []*client.ExternalAccount{},
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
			Expect(state.NextCursor).To(BeEmpty())
		})

		It("should fetch next external accounts - pageSize is total accounts", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 2,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.ExternalAccount]{
				Data: sampleExternalAccounts,
			})

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.NextCursor).To(BeEmpty())
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
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
			).SetArg(2, client.ResponseWrapper[[]*client.ExternalAccount]{
				Data: sampleExternalAccounts,
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
			Expect(state.NextCursor).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
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
			).SetArg(2, client.ResponseWrapper[[]*client.ExternalAccount]{
				Data:       sampleExternalAccounts[:40],
				NextCursor: "qwerty",
			})

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.NextCursor).To(Equal("qwerty"))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
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
			).SetArg(2, client.ResponseWrapper[[]*client.ExternalAccount]{
				Data:       sampleExternalAccounts[:40],
				NextCursor: "asdfg",
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
			Expect(state.NextCursor).To(Equal("asdfg"))
		})
	})
})
