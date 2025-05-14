package bankingcircle

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("BankingCircle Plugin Bank Account Creation", func() {
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
			sampleBankAccount models.BankAccount
			now               time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleBankAccount = models.BankAccount{
				ID:            uuid.New(),
				CreatedAt:     now.UTC(),
				Name:          "test",
				AccountNumber: pointer.For("123456789"),
				Country:       pointer.For("US"),
				Metadata: map[string]string{
					"test": "test",
				},
			}
		})

		It("should create bank account", func(ctx SpecContext) {
			resp, err := plg.CreateBankAccount(ctx, models.CreateBankAccountRequest{
				BankAccount: sampleBankAccount,
			})

			raw, _ := json.Marshal(sampleBankAccount)

			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: sampleBankAccount.ID.String(),
					CreatedAt: now.UTC(),
					Name:      pointer.For("test"),
					Metadata:  sampleBankAccount.Metadata,
					Raw:       raw,
				},
			}))
		})
	})
})
