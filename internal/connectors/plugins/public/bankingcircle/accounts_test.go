package bankingcircle

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Currencycloud Plugin Accounts", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetch next accounts", func() {
		var (
			m              *client.MockClient
			sampleAccounts []client.Account
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now()

			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, client.Account{
					AccountID:          fmt.Sprintf("%d", i),
					AccountDescription: fmt.Sprintf("account-%d", i),
					Currency:           "EUR",
					OpeningDate:        now.Add(-time.Duration(50-i) * time.Minute).Format("2006-01-02T15:04:05.999999999+00:00"),
				})
			}
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetAccounts(ctx, 1, 60, time.Time{}).Return(
				[]client.Account{},
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
			Expect(state.LastAccountID).To(BeEmpty())
			Expect(state.FromOpeningDate.IsZero()).To(BeTrue())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			lastOpeningDate, _ := time.Parse("2006-01-02T15:04:05.999999999+00:00", sampleAccounts[49].OpeningDate)

			m.EXPECT().GetAccounts(ctx, 1, 60, time.Time{}).Return(
				sampleAccounts,
				nil,
			)

			m.EXPECT().GetAccounts(ctx, 2, 60, time.Time{}).Return(
				[]client.Account{},
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
			Expect(state.LastAccountID).To(Equal("49"))
			Expect(state.FromOpeningDate.UTC()).To(Equal(lastOpeningDate.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(ctx, gomock.Any(), 40, time.Time{}).Return(
				sampleAccounts[:40],
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
			Expect(state.LastAccountID).To(Equal("39"))
			lastOpeningDate, _ := time.Parse("2006-01-02T15:04:05.999999999+00:00", sampleAccounts[39].OpeningDate)
			Expect(state.FromOpeningDate.UTC()).To(Equal(lastOpeningDate.UTC()))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			lastOpeningDate, _ := time.Parse("2006-01-02T15:04:05.999999999+00:00", sampleAccounts[38].OpeningDate)
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastAccountID": "%s", "fromOpeningDate": "%s"}`, sampleAccounts[38].AccountID, lastOpeningDate.Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(ctx, 1, 40, lastOpeningDate).Return(
				sampleAccounts[:40],
				nil,
			)

			m.EXPECT().GetAccounts(ctx, 2, 40, lastOpeningDate).Return(
				sampleAccounts[41:],
				nil,
			)

			m.EXPECT().GetAccounts(ctx, 3, 40, lastOpeningDate).Return(
				[]client.Account{},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastAccountID).To(Equal("49"))
			lastOpeningDate, _ = time.Parse("2006-01-02T15:04:05.999999999+00:00", sampleAccounts[49].OpeningDate)
			Expect(state.FromOpeningDate.UTC()).To(Equal(lastOpeningDate.UTC()))
		})
	})
})
