package bankingcircle

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("BankingCircle Plugin Bank Account Creation", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next accounts", func() {
		var (
			m                  *client.MockClient
			sampleBankAccount  models.BankAccount
			sampleCounterParty models.PSPCounterParty
			now                time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
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

			sampleCounterParty = models.PSPCounterParty{
				ID:          uuid.New(),
				Name:        "test",
				CreatedAt:   now.UTC(),
				BankAccount: &sampleBankAccount,
				Metadata: map[string]string{
					"test": "test",
				},
			}
		})

		It("should create bank account from bank account", func(ctx SpecContext) {
			resp, err := plg.CreateBankAccount(ctx, models.CreateBankAccountRequest{
				BankAccount: &sampleBankAccount,
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

		It("should create bank account from counter party", func(ctx SpecContext) {
			resp, err := plg.CreateBankAccount(ctx, models.CreateBankAccountRequest{
				CounterParty: &sampleCounterParty,
			})

			raw, _ := json.Marshal(sampleCounterParty)

			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: sampleCounterParty.ID.String(),
					CreatedAt: now.UTC(),
					Name:      pointer.For("test"),
					Metadata:  sampleCounterParty.Metadata,
					Raw:       raw,
				},
			}))
		})
	})
})
