package adyen

import (
	"encoding/json"
	"fmt"

	"github.com/adyen/adyen-go-api-library/v7/src/management"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/adyen/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Adyen Plugin Accounts", func() {
	var (
		m   *client.MockClient
		plg models.Plugin
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)

		plg = &Plugin{client: m}
	})

	Context("fetching next accounts", func() {
		var (
			sampleAccounts []management.Merchant
		)

		BeforeEach(func() {

			for i := 10; i < 60; i++ {
				sampleAccounts = append(sampleAccounts, management.Merchant{
					Id:   pointer.For(fmt.Sprintf("%d", i)),
					Name: pointer.For(fmt.Sprintf("name-%d", i)),
				})
			}
		})

		AfterEach(func() {
			sampleAccounts = nil
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetMerchantAccounts(gomock.Any(), int32(1), int32(60)).Return(
				[]management.Merchant{},
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
			Expect(state.LastPage).To(Equal(0))
			Expect(state.LastID).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetMerchantAccounts(gomock.Any(), int32(1), int32(60)).Return(
				sampleAccounts,
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
			Expect(state.LastPage).To(Equal(0))
			Expect(state.LastID).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetMerchantAccounts(gomock.Any(), int32(1), int32(40)).Return(
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
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastID).To(Equal(*sampleAccounts[39].Id))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastPage": %d, "lastId": "%s"}`, 1, *sampleAccounts[38].Id)),
				PageSize: 40,
			}

			m.EXPECT().GetMerchantAccounts(gomock.Any(), int32(1), int32(40)).Return(
				sampleAccounts[:40],
				nil,
			)

			m.EXPECT().GetMerchantAccounts(gomock.Any(), int32(2), int32(40)).Return(
				sampleAccounts[41:],
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
			// We fetched everything, state should be resetted
			Expect(state.LastPage).To(Equal(0))
			Expect(state.LastID).To(BeEmpty())
		})
	})
})
