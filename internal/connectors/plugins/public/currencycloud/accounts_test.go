package currencycloud

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
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
			sampleAccounts []*client.Account
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now()

			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, &client.Account{
					ID:          fmt.Sprintf("%d", i),
					AccountName: fmt.Sprintf("account-%d", i),
					CreatedAt:   now.Add(time.Duration(50-i) * time.Minute),
					UpdatedAt:   now.Add(time.Duration(50-i) * time.Minute),
				})
			}
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetAccounts(ctx, gomock.Any(), 60).Return(
				sampleAccounts,
				-1,
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
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreatedAt.UTC()).To(Equal(sampleAccounts[49].CreatedAt.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetAccounts(ctx, gomock.Any(), 40).Return(
				sampleAccounts[:40],
				2,
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
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreatedAt.UTC()).To(Equal(sampleAccounts[39].CreatedAt.UTC()))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastPage": %d, "lastCreatedAt": "%s"}`, 1, sampleAccounts[39].CreatedAt)),
				PageSize: 40,
			}
		})
	})
})
