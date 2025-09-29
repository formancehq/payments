package tink

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Balances", func() {
	Context("fetchNextBalances", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should fetch balances successfully", func(ctx SpecContext) {
			userID := "user-123"
			accountID := "acc-456"

			webhookPayload := fetchNextDataRequest{
				UserID:         userID,
				ExternalUserID: userID,
				AccountID:      accountID,
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			fromPayload := models.OpenBankingForwardedUserFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			refTime := time.Now().UTC().Truncate(time.Second)
			resp := client.AccountBalanceResponse{
				AccountId: accountID,
				Refreshed: refTime,
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						CurrencyCode:     "EUR",
						ValueInMinorUnit: json.Number("12345"),
						Value: client.AccountBalanceValue{
							Scale:         json.Number("2"),
							UnscaledValue: json.Number("12345"),
						},
					},
				},
			}

			m.EXPECT().GetAccountBalances(gomock.Any(), userID, accountID).Return(resp, nil)

			req := models.FetchNextBalancesRequest{FromPayload: fromPayloadBytes}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(out.Balances).To(HaveLen(1))

			b := out.Balances[0]
			Expect(b.AccountReference).To(Equal(accountID))
			Expect(b.CreatedAt.UTC()).To(Equal(refTime))
			Expect(b.Asset).To(Equal("EUR/2"))
			Expect(b.Amount.Cmp(big.NewInt(12345))).To(Equal(0))
		})

		It("should handle client error", func(ctx SpecContext) {
			userID := "user-123"
			accountID := "acc-456"
			webhookPayload := fetchNextDataRequest{UserID: userID, ExternalUserID: userID, AccountID: accountID}
			wpb, _ := json.Marshal(webhookPayload)
			fp := models.OpenBankingForwardedUserFromPayload{FromPayload: wpb}
			fpb, _ := json.Marshal(fp)

			m.EXPECT().GetAccountBalances(gomock.Any(), userID, accountID).Return(client.AccountBalanceResponse{}, errors.New("client error"))

			req := models.FetchNextBalancesRequest{FromPayload: fpb}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(out).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should handle invalid outer from payload", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{FromPayload: []byte("not json")}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(out).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should handle invalid inner webhook payload", func(ctx SpecContext) {
			fromPayloadBytes := []byte(`{"fromPayload": "invalid json"}`)
			req := models.FetchNextBalancesRequest{FromPayload: fromPayloadBytes}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(out).To(Equal(models.FetchNextBalancesResponse{}))
		})
	})

	Context("toPSPBalance", func() {
		It("should convert balance correctly", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			resp := client.AccountBalanceResponse{
				AccountId: "acc-1",
				Refreshed: refTime,
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						CurrencyCode:     "USD",
						ValueInMinorUnit: json.Number("1000"),
						Value:            client.AccountBalanceValue{Scale: json.Number("2"), UnscaledValue: json.Number("1000")},
					},
				},
			}

			psp, err := toPSPBalance(resp, "acc-1")
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-1"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("USD/2"))
			Expect(psp.Amount.Cmp(big.NewInt(1000))).To(Equal(0))
		})

		It("should error on invalid amount", func() {
			resp := client.AccountBalanceResponse{
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{Value: client.AccountBalanceValue{Scale: json.Number("2"), UnscaledValue: json.Number("not-a-number")}},
				},
			}
			_, err := toPSPBalance(resp, "acc")
			Expect(err).ToNot(BeNil())
		})
	})
})
