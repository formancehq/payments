package chainbridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/chainbridge/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("ChainBridge Plugin Accounts", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
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
			sampleMonitors []*client.Monitor
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleMonitors = make([]*client.Monitor, 0)
			for i := 0; i < 5; i++ {
				sampleMonitors = append(sampleMonitors, &client.Monitor{
					ID:        fmt.Sprintf("mon_%d", i),
					Chain:     "ethereum",
					Address:   fmt.Sprintf("0x%040d", i),
					Status:    "active",
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC(),
				})
			}
		})

		It("should return an error - get monitors error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetMonitors(gomock.Any()).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetMonitors(gomock.Any()).Return(
				[]*client.Monitor{},
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
			Expect(state.LastCreatedAt.IsZero()).To(BeTrue())
		})

		It("should fetch all accounts - no state", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetMonitors(gomock.Any()).Return(
				sampleMonitors,
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(5))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastCreatedAt).To(Equal(sampleMonitors[4].CreatedAt))

			// Verify account mapping
			Expect(resp.Accounts[0].Reference).To(Equal("mon_0"))
			Expect(*resp.Accounts[0].Name).To(Equal(fmt.Sprintf("0x%040d", 0)))
			Expect(resp.Accounts[0].Metadata["chain"]).To(Equal("ethereum"))
			Expect(resp.Accounts[0].Metadata["address"]).To(Equal(fmt.Sprintf("0x%040d", 0)))
			Expect(resp.Accounts[0].Metadata["status"]).To(Equal("active"))
		})

		It("should skip already ingested accounts with state", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastCreatedAt": "%s"}`, sampleMonitors[2].CreatedAt.Format(time.RFC3339Nano))),
				PageSize: 60,
			}

			m.EXPECT().GetMonitors(gomock.Any()).Return(
				sampleMonitors,
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.Accounts[0].Reference).To(Equal("mon_3"))
			Expect(resp.Accounts[1].Reference).To(Equal("mon_4"))

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastCreatedAt).To(Equal(sampleMonitors[4].CreatedAt))
		})
	})
})
