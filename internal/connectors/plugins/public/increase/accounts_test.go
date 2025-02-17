package increase

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
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
			m              *client.MockClient
			sampleAccounts []*client.Account
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
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
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{invalid json`),
				PageSize: 10,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("invalid character 'i' looking for beginning of object key string"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should return an error - invalid created_at time", func(ctx SpecContext) {
			accounts := []*client.Account{{
				ID:        "acc_123",
				Name:      "Test Account",
				Currency:  "USD",
				CreatedAt: "invalid-timestamp",
				Type:      "CHECKING",
				Bank:      "test_bank",
				Status:    "ACTIVE",
			}}
		
			resp, err := plg.fillAccounts(accounts, make([]models.PSPAccount, 0), 10)
		
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(`parsing time "invalid-timestamp" as "2006-01-02T15:04:05Z07:00": cannot parse "invalid-timestamp" as "2006"`))
			Expect(resp).To(BeNil())
		})		

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 60, "", time.Time{}).Return(
				[]*client.Account{},
				"",
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

			m.EXPECT().GetAccounts(gomock.Any(), 60, "", time.Time{}).Return(
				[]*client.Account{},
				"",
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
			Expect(state.NextCursor).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 60, "", time.Time{}).Return(
				sampleAccounts,
				"",
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
			Expect(state.NextCursor).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 40, "", time.Time{}).Return(
				sampleAccounts[:40],
				"wrY4nKh",
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
			Expect(state.NextCursor).To(Equal("wrY4nKh"))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			createdAtAfter, _ := time.Parse(time.RFC3339, sampleAccounts[38].CreatedAt)
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"next_cursor": "wrY4nKh", "created_at_after": "%s"}`, createdAtAfter.UTC().Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(gomock.Any(), 40, "wrY4nKh", createdAtAfter.UTC()).Return(
				sampleAccounts[:40],
				"qsdf",
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
			// We fetched everything, state should be resetted
			Expect(state.NextCursor).To(Equal("qsdf"))
		})
	})
})
