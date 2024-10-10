package moneycorp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp Plugin ExternalAccounts", func() {
	var (
		plg *Plugin
	)

	Context("fetch next ExternalAccounts", func() {
		var (
			m *client.MockClient

			pageSize               int
			sampleExternalAccounts []*client.Recipient
			accRef                 string
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			pageSize = 10
			accRef = "baseAcc"
			sampleExternalAccounts = make([]*client.Recipient, 0)
			for i := 0; i < pageSize; i++ {
				sampleExternalAccounts = append(sampleExternalAccounts, &client.Recipient{
					Attributes: client.RecipientAttributes{
						BankAccountCurrency: "JPY",
						CreatedAt:           strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
						BankAccountName:     "jpy account",
					},
				})
			}

		})
		It("fetches next ExternalAccounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, accRef)),
				State:       json.RawMessage(`{}`),
				PageSize:    pageSize,
			}
			m.EXPECT().GetRecipients(ctx, accRef, gomock.Any(), pageSize).Return(
				sampleExternalAccounts,
				nil,
			)
			res, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.ExternalAccounts).To(HaveLen(pageSize))
			Expect(*res.ExternalAccounts[0].Name).To(Equal(sampleExternalAccounts[0].Attributes.BankAccountName))

			var state accountsState

			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastPage).To(Equal(0))
		})
	})
})
