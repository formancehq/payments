package moneycorp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp *Plugin ExternalAccounts", func() {
	var (
		plg models.Plugin
	)

	Context("fetch next ExternalAccounts", func() {
		var (
			ctrl *gomock.Controller
			m    *client.MockClient

			sampleExternalAccounts []*client.Recipient
			accRef                 string
			now                    time.Time
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
			accRef = "baseAcc"
			now = time.Now().UTC()

			sampleExternalAccounts = make([]*client.Recipient, 0)
			for i := 0; i < 50; i++ {
				sampleExternalAccounts = append(sampleExternalAccounts, &client.Recipient{
					Attributes: client.RecipientAttributes{
						BankAccountCurrency: "JPY",
						CreatedAt:           strings.TrimSuffix(now.Add(-time.Duration(60-i)*time.Minute).UTC().Format(time.RFC3339Nano), "Z"),
						BankAccountName:     "jpy account",
					},
				})
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - missing from payload", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing from payload in request"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 60).Return(
				[]*client.Recipient{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 60).Return(
				[]*client.Recipient{},
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
			Expect(state.LastPage).To(Equal(0))
			Expect(state.LastCreatedAt.IsZero()).To(BeTrue())
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 60).Return(
				sampleExternalAccounts,
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
			Expect(state.LastPage).To(Equal(0))
			createdAtTime, _ := time.Parse(time.RFC3339Nano, sampleExternalAccounts[49].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(`{}`),
				PageSize:    40,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 40).Return(
				sampleExternalAccounts[:40],
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
			Expect(state.LastPage).To(Equal(0))
			createdAtTime, _ := time.Parse(time.RFC3339Nano, sampleExternalAccounts[39].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			lastCreaatedAt, _ := time.Parse(time.RFC3339Nano, sampleExternalAccounts[38].Attributes.CreatedAt+"Z")
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(fmt.Sprintf(`{"lastPage": %d, "lastCreatedAt": "%s"}`, 0, lastCreaatedAt.Format(time.RFC3339Nano))),
				PageSize:    40,
				FromPayload: []byte(`{"reference": "baseAcc"}`),
			}

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 0, 40).Return(
				sampleExternalAccounts[:40],
				nil,
			)

			m.EXPECT().GetRecipients(gomock.Any(), accRef, 1, 40).Return(
				sampleExternalAccounts[41:],
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
			Expect(state.LastPage).To(Equal(1))
			createdAtTime, _ := time.Parse(time.RFC3339Nano, sampleExternalAccounts[49].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})
	})
})
