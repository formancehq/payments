package tink

import (
	"encoding/json"
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
			accountID := "acc-456"

			refTime := time.Now().UTC().Truncate(time.Second)
			account := client.Account{
				ID:   accountID,
				Name: "Test Account",
				Type: "CHECKING",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "EUR",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "2",
								Value: "12345",
							},
						},
					},
				},
				Dates: client.Dates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := models.PSPAccount{
				Raw: accountBytes,
			}
			pspAccountBytes, err := json.Marshal(pspAccount)
			Expect(err).To(BeNil())

			req := models.FetchNextBalancesRequest{FromPayload: pspAccountBytes}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(out.Balances).To(HaveLen(1))

			b := out.Balances[0]
			Expect(b.AccountReference).To(Equal(accountID))
			Expect(b.CreatedAt.UTC()).To(Equal(refTime))
			Expect(b.Asset).To(Equal("EUR/2"))
			Expect(b.Amount.Cmp(big.NewInt(12345))).To(Equal(0))
		})

		It("should handle invalid PSPAccount payload", func(ctx SpecContext) {
			invalidAccountBytes := []byte("invalid json")
			pspAccount := models.PSPAccount{
				Raw: invalidAccountBytes,
			}
			pspAccountBytes, _ := json.Marshal(pspAccount)

			req := models.FetchNextBalancesRequest{FromPayload: pspAccountBytes}
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

		It("should handle invalid inner PSPAccount payload", func(ctx SpecContext) {
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
			account := client.Account{
				ID:   "acc-1",
				Name: "Test Account",
				Type: "CHECKING",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "USD",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "2",
								Value: "1000",
							},
						},
					},
				},
				Dates: client.Dates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := models.PSPAccount{
				Raw: accountBytes,
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-1"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("USD/2"))
			Expect(psp.Amount.Cmp(big.NewInt(1000))).To(Equal(0))
		})

		It("should error on invalid amount", func() {
			account := client.Account{
				ID: "acc",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "USD",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "2",
								Value: "not-a-number",
							},
						},
					},
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := models.PSPAccount{
				Raw: accountBytes,
			}

			_, err = toPSPBalance(pspAccount)
			Expect(err).ToNot(BeNil())
		})

		It("should error on unsupported currency", func() {
			account := client.Account{
				ID: "acc",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "INVALID",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "2",
								Value: "1000",
							},
						},
					},
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := models.PSPAccount{
				Raw: accountBytes,
			}

			_, err = toPSPBalance(pspAccount)
			Expect(err).ToNot(BeNil())
		})

		It("should handle zero amount", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			account := client.Account{
				ID:   "acc-zero",
				Name: "Zero Balance Account",
				Type: "CHECKING",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "EUR",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "2",
								Value: "0",
							},
						},
					},
				},
				Dates: client.Dates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := models.PSPAccount{
				Raw: accountBytes,
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-zero"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("EUR/2"))
			Expect(psp.Amount.Cmp(big.NewInt(0))).To(Equal(0))
		})

		It("should handle negative amount", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			account := client.Account{
				ID:   "acc-negative",
				Name: "Negative Balance Account",
				Type: "CHECKING",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "USD",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "2",
								Value: "-5000",
							},
						},
					},
				},
				Dates: client.Dates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := models.PSPAccount{
				Raw: accountBytes,
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-negative"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("USD/2"))
			Expect(psp.Amount.Cmp(big.NewInt(-5000))).To(Equal(0))
		})

		It("should handle different currency precisions", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			account := client.Account{
				ID:   "acc-jpy",
				Name: "JPY Account",
				Type: "CHECKING",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "JPY",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "0",
								Value: "100000",
							},
						},
					},
				},
				Dates: client.Dates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := models.PSPAccount{
				Raw: accountBytes,
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-jpy"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("JPY/0"))
			Expect(psp.Amount.Cmp(big.NewInt(100000))).To(Equal(0))
		})

		It("should error on invalid JSON in PSPAccount", func() {
			pspAccount := models.PSPAccount{
				Raw: []byte("invalid json"),
			}

			_, err := toPSPBalance(pspAccount)
			Expect(err).ToNot(BeNil())
		})
	})
})
