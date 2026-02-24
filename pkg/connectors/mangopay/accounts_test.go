package mangopay

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/pkg/connectors/mangopay/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin Accounts", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  connector.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next accounts", func() {
		var (
			sampleAccounts []client.Wallet
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleAccounts = make([]client.Wallet, 0)
			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, client.Wallet{
					ID:           fmt.Sprintf("%d", i),
					Description:  fmt.Sprintf("Account %d", i),
					CreationDate: now.Add(-time.Duration(50-i) * time.Minute).UTC().Unix(),
					Currency:     "USD",
				})
			}
		})

		It("should return an error - missing from payload", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
				PageSize: 60,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing from payload in request"))
			Expect(resp).To(Equal(connector.FetchNextAccountsResponse{}))
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetWallets(gomock.Any(), "test", 1, 60).Return(
				[]client.Wallet{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(connector.FetchNextAccountsResponse{}))
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetWallets(gomock.Any(), "test", 1, 60).Return(
				[]client.Wallet{},
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
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreationDate.IsZero()).To(BeTrue())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetWallets(gomock.Any(), "test", 1, 60).Return(
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
			createdTime := time.Unix(sampleAccounts[49].CreationDate, 0)
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
				PageSize:    40,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetWallets(gomock.Any(), "test", 1, 40).Return(
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
			createdTime := time.Unix(sampleAccounts[39].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			lastCreatedAt := time.Unix(sampleAccounts[39].CreationDate, 0)
			req := connector.FetchNextAccountsRequest{
				State:       []byte(fmt.Sprintf(`{"lastPage": 1, "lastCreationDate": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano))),
				PageSize:    40,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetWallets(gomock.Any(), "test", 1, 40).Return(
				sampleAccounts[:40],
				nil,
			)

			m.EXPECT().GetWallets(gomock.Any(), "test", 2, 40).Return(
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
			Expect(state.LastPage).To(Equal(2))
			createdTime := time.Unix(sampleAccounts[49].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next accounts - when lastCreationDate form last page is equal to one of the new page's", func(ctx SpecContext) {
			lastCreatedAt := time.Unix(sampleAccounts[9].CreationDate, 0)
			sampleAccounts[10].CreationDate = sampleAccounts[9].CreationDate
			req := connector.FetchNextAccountsRequest{
				State:       []byte(fmt.Sprintf(`{"lastPage": 2, "lastCreationDate": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano))),
				PageSize:    10,
				FromPayload: json.RawMessage(`{"Id": "test"}`),
			}

			m.EXPECT().GetWallets(gomock.Any(), "test", 2, 10).Times(1).Return(
				sampleAccounts[10:20],
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(10))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastPage).To(Equal(2))
			createdTime := time.Unix(sampleAccounts[19].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})
	})
})
