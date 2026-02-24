package currencycloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/pkg/connectors/currencycloud/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("CurrencyCloud Plugin External Accounts", func() {
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

	Context("fetching next external accounts", func() {
		var (
			sampleBeneficiaries []*client.Beneficiary
			now                 time.Time
		)

		BeforeEach(func() {

			now = time.Now().UTC()

			sampleBeneficiaries = make([]*client.Beneficiary, 0)
			for i := 0; i < 50; i++ {
				sampleBeneficiaries = append(sampleBeneficiaries, &client.Beneficiary{
					ID:                    fmt.Sprintf("%d", i),
					BankAccountHolderName: fmt.Sprintf("Account %d", i),
					Name:                  fmt.Sprintf("Account %d", i),
					Currency:              "EUR",
					CreatedAt:             now.Add(-time.Duration(50-i) * time.Minute).UTC(),
				})
			}
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 60).Return(
				[]*client.Beneficiary{},
				-1,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(connector.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 60).Return(
				[]*client.Beneficiary{},
				-1,
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
			Expect(state.LastCreatedAt.IsZero()).To(BeTrue())
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 60).Return(
				sampleBeneficiaries,
				-1,
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
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreatedAt).To(Equal(sampleBeneficiaries[49].CreatedAt))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 40).Return(
				sampleBeneficiaries[:40],
				2,
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
			Expect(state.LastCreatedAt).To(Equal(sampleBeneficiaries[39].CreatedAt))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastPage": %d, "lastCreatedAt": "%s"}`, 1, sampleBeneficiaries[38].CreatedAt.Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), 1, 40).Return(
				sampleBeneficiaries[:40],
				2,
				nil,
			)

			m.EXPECT().GetBeneficiaries(gomock.Any(), 2, 40).Return(
				sampleBeneficiaries[41:],
				-1,
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
			Expect(state.LastCreatedAt).To(Equal(sampleBeneficiaries[49].CreatedAt))
		})
	})
})
