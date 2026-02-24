package tink

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Balances", func() {
	Context("fetchNextBalances", func() {
		var (
			ctrl *gomock.Controller
			plg  connector.Plugin
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
			psuID := uuid.MustParse("d5d4a5e1-1f02-4a5f-b9b5-518232fde991")

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
				Dates: client.AccountDates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw:   accountBytes,
				PsuID: &psuID,
			}
			pspAccountBytes, err := json.Marshal(pspAccount)
			Expect(err).To(BeNil())

			req := connector.FetchNextBalancesRequest{FromPayload: pspAccountBytes}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(out.Balances).To(HaveLen(1))

			b := out.Balances[0]
			Expect(b.AccountReference).To(Equal(accountID))
			Expect(b.CreatedAt.UTC()).To(Equal(refTime))
			Expect(b.Asset).To(Equal("EUR/2"))
			Expect(b.Amount.Cmp(big.NewInt(12345))).To(Equal(0))
			Expect(b.PsuID).ToNot(BeNil())
			Expect(b.PsuID.String()).To(Equal(psuID.String()))
		})

		It("should handle invalid PSPAccount payload", func(ctx SpecContext) {
			invalidAccountBytes := []byte("invalid json")
			pspAccount := connector.PSPAccount{
				Raw: invalidAccountBytes,
			}
			pspAccountBytes, _ := json.Marshal(pspAccount)

			req := connector.FetchNextBalancesRequest{FromPayload: pspAccountBytes}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(out).To(Equal(connector.FetchNextBalancesResponse{}))
		})

		It("should handle invalid outer from payload", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{FromPayload: []byte("not json")}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(out).To(Equal(connector.FetchNextBalancesResponse{}))
		})

		It("should handle invalid inner PSPAccount payload", func(ctx SpecContext) {
			fromPayloadBytes := []byte(`{"fromPayload": "invalid json"}`)
			req := connector.FetchNextBalancesRequest{FromPayload: fromPayloadBytes}
			out, err := plg.(*Plugin).fetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(out).To(Equal(connector.FetchNextBalancesResponse{}))
		})
	})

	Context("toPSPBalance", func() {
		It("should convert balance correctly", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			psuID := uuid.MustParse("a1b2c3d4-5e6f-4788-1234-567890abcdef")
			openBankingConnectionID := "connection-123"

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
				Dates: client.AccountDates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw:                     accountBytes,
				PsuID:                   &psuID,
				OpenBankingConnectionID: &openBankingConnectionID,
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-1"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("USD/2"))
			Expect(psp.Amount.Cmp(big.NewInt(1000))).To(Equal(0))
			Expect(psp.PsuID).ToNot(BeNil())
			Expect(psp.PsuID.String()).To(Equal(psuID.String()))
			Expect(psp.OpenBankingConnectionID).ToNot(BeNil())
			Expect(*psp.OpenBankingConnectionID).To(Equal(openBankingConnectionID))
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

			pspAccount := connector.PSPAccount{
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

			pspAccount := connector.PSPAccount{
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
				Dates: client.AccountDates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
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
				Dates: client.AccountDates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
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
				Dates: client.AccountDates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
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
			pspAccount := connector.PSPAccount{
				Raw: []byte("invalid json"),
			}

			_, err := toPSPBalance(pspAccount)
			Expect(err).ToNot(BeNil())
		})

		It("should populate PSUID and OpenBankingConnectionID from PSPAccount", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			psuID := uuid.MustParse("f1e2d3c4-b5a6-9788-1234-567890abcdef")
			openBankingConnectionID := "tink-connection-456"

			account := client.Account{
				ID:   "acc-psu-test",
				Name: "PSU Test Account",
				Type: "SAVINGS",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "GBP",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "2",
								Value: "5000",
							},
						},
					},
				},
				Dates: client.AccountDates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw:                     accountBytes,
				PsuID:                   &psuID,
				OpenBankingConnectionID: &openBankingConnectionID,
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-psu-test"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("GBP/2"))
			Expect(psp.Amount.Cmp(big.NewInt(5000))).To(Equal(0))

			// Verify PSUID is populated correctly
			Expect(psp.PsuID).ToNot(BeNil())
			Expect(psp.PsuID.String()).To(Equal(psuID.String()))

			// Verify OpenBankingConnectionID is populated correctly
			Expect(psp.OpenBankingConnectionID).ToNot(BeNil())
			Expect(*psp.OpenBankingConnectionID).To(Equal(openBankingConnectionID))
		})

		It("should handle missing PSUID and OpenBankingConnectionID", func() {
			refTime := time.Now().UTC().Truncate(time.Second)

			account := client.Account{
				ID:   "acc-no-psu",
				Name: "No PSU Account",
				Type: "CHECKING",
				Balances: client.AccountBalances{
					Booked: client.AccountBalance{
						Amount: client.Amount{
							CurrencyCode: "CAD",
							Value: struct {
								Scale string `json:"scale"`
								Value string `json:"unscaledValue"`
							}{
								Scale: "2",
								Value: "2500",
							},
						},
					},
				},
				Dates: client.AccountDates{
					LastRefreshed: refTime,
				},
			}

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw: accountBytes,
				// No PsuID or OpenBankingConnectionID set
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-no-psu"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("CAD/2"))
			Expect(psp.Amount.Cmp(big.NewInt(2500))).To(Equal(0))

			// Verify PSUID is nil when not provided
			Expect(psp.PsuID).To(BeNil())

			// Verify OpenBankingConnectionID is nil when not provided
			Expect(psp.OpenBankingConnectionID).To(BeNil())
		})
	})
})
