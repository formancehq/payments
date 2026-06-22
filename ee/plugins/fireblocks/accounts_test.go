package fireblocks

import (
	"encoding/json"
	"time"

	"github.com/formancehq/payments/ee/plugins/fireblocks/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Fireblocks Plugin Accounts", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("fetches next accounts with cursor and surfaces vault metadata", func(ctx SpecContext) {
		state, err := json.Marshal(accountsState{NextCursor: "cursor-1"})
		Expect(err).To(BeNil())

		creationDate := int64(1700000000000)
		m.EXPECT().GetVaultAccountsPaged(gomock.Any(), "cursor-1", 2).Return(&client.VaultAccountsPagedResponse{
			Accounts: []client.VaultAccount{
				{
					ID:            "acc-1",
					Name:          "Treasury",
					CustomerRefID: "cust-42",
					HiddenOnUI:    true,
					AutoFuel:      true,
					CreationDate:  creationDate,
				},
				{
					ID:           "acc-2",
					Name:         "Ops",
					CreationDate: creationDate + 1000,
				},
			},
			Paging: client.Paging{After: "next"},
		}, nil)

		resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{
			State:    state,
			PageSize: 2,
		})
		Expect(err).To(BeNil())
		Expect(resp.Accounts).To(HaveLen(2))
		Expect(resp.HasMore).To(BeTrue())

		Expect(resp.Accounts[0].Reference).To(Equal("acc-1"))
		Expect(*resp.Accounts[0].Name).To(Equal("Treasury"))
		Expect(resp.Accounts[0].CreatedAt).To(Equal(time.Unix(1700000000, 0)))
		Expect(resp.Accounts[0].Raw).ToNot(BeNil())
		Expect(resp.Accounts[0].Metadata).To(HaveKeyWithValue(MetadataPrefix+"customer_ref_id", "cust-42"))
		Expect(resp.Accounts[0].Metadata).To(HaveKeyWithValue(MetadataPrefix+"hidden_on_ui", "true"))
		Expect(resp.Accounts[0].Metadata).To(HaveKeyWithValue(MetadataPrefix+"auto_fuel", "true"))

		// Defaults: no metadata emitted when source fields are empty / false.
		Expect(resp.Accounts[1].Metadata).To(BeNil())

		var newState accountsState
		err = json.Unmarshal(resp.NewState, &newState)
		Expect(err).To(BeNil())
		Expect(newState.NextCursor).To(Equal("next"))
	})
})
