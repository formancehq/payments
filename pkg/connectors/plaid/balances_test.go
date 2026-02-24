package plaid

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/plaid/plaid-go/v34/plaid"
)

var _ = Describe("Plaid *Plugin Balances", func() {
	Context("fetchNextBalances", func() {
		var (
			plg connector.Plugin
		)

		BeforeEach(func() {
			plg = &Plugin{}
		})

		It("should fetch balances successfully", func(ctx SpecContext) {
			accountID := "acc-456"
			psuID := uuid.MustParse("d5d4a5e1-1f02-4a5f-b9b5-518232fde991")
			openBankingConnectionID := "plaid-connection-123"

			refTime := time.Now().UTC().Truncate(time.Second)

			// Create a Plaid account with balance
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId(accountID)
			account.SetName("Test Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(123.45)
			balance.SetIsoCurrencyCode("USD")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw:                     accountBytes,
				PsuID:                   &psuID,
				OpenBankingConnectionID: &openBankingConnectionID,
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
			Expect(b.Asset).To(Equal("USD/2"))
			Expect(b.Amount).To(Equal(big.NewInt(12345)))
			Expect(b.PsuID).ToNot(BeNil())
			Expect(b.PsuID.String()).To(Equal(psuID.String()))
			Expect(b.OpenBankingConnectionID).ToNot(BeNil())
			Expect(*b.OpenBankingConnectionID).To(Equal(openBankingConnectionID))
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
			openBankingConnectionID := "plaid-connection-123"

			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-1")
			account.SetName("Test Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(1000.50)
			balance.SetIsoCurrencyCode("USD")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

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
			Expect(psp.Amount).To(Equal(big.NewInt(100050)))
			Expect(psp.PsuID).ToNot(BeNil())
			Expect(psp.PsuID.String()).To(Equal(psuID.String()))
			Expect(psp.OpenBankingConnectionID).ToNot(BeNil())
			Expect(*psp.OpenBankingConnectionID).To(Equal(openBankingConnectionID))
		})

		It("should error on unsupported currency", func() {
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc")

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(1000.0)
			balance.SetIsoCurrencyCode("INVALID")
			account.SetBalances(*balance)

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw: accountBytes,
			}

			_, err = toPSPBalance(pspAccount)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing currencies"))
		})

		It("should handle zero amount", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-zero")
			account.SetName("Zero Balance Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(0.0)
			balance.SetIsoCurrencyCode("EUR")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

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
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-negative")
			account.SetName("Negative Balance Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(-50.25)
			balance.SetIsoCurrencyCode("USD")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

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
			Expect(psp.Amount).To(Equal(big.NewInt(-5025)))
		})

		It("should handle different currency precisions", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-jpy")
			account.SetName("JPY Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(100000.0)
			balance.SetIsoCurrencyCode("JPY")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

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
			Expect(psp.Amount).To(Equal(big.NewInt(100000)))
		})

		It("should error on invalid JSON in PSPAccount", func() {
			pspAccount := connector.PSPAccount{
				Raw: []byte("invalid json"),
			}

			_, err := toPSPBalance(pspAccount)
			Expect(err).ToNot(BeNil())
		})

		It("should error when balance is not set", func() {
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-no-balance")

			// Create balance without setting current value
			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetIsoCurrencyCode("USD")
			account.SetBalances(*balance)

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw: accountBytes,
			}

			_, err = toPSPBalance(pspAccount)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("balance is not set"))
		})

		It("should handle unofficial currency code when ISO code is not set", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-unofficial")
			account.SetName("Unofficial Currency Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(500.0)
			balance.SetUnofficialCurrencyCode("BTC")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw: accountBytes,
			}

			_, err = toPSPBalance(pspAccount)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing currencies"))
		})

		It("should populate PSUID and OpenBankingConnectionID from PSPAccount", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			psuID := uuid.MustParse("f1e2d3c4-b5a6-9788-1234-567890abcdef")
			openBankingConnectionID := "plaid-connection-456"

			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-psu-test")
			account.SetName("PSU Test Account")
			account.SetType(plaid.ACCOUNTTYPE_CREDIT)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(5000.0)
			balance.SetIsoCurrencyCode("GBP")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

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
			Expect(psp.Amount).To(Equal(big.NewInt(500000)))

			// Verify PSUID is populated correctly
			Expect(psp.PsuID).ToNot(BeNil())
			Expect(psp.PsuID.String()).To(Equal(psuID.String()))

			// Verify OpenBankingConnectionID is populated correctly
			Expect(psp.OpenBankingConnectionID).ToNot(BeNil())
			Expect(*psp.OpenBankingConnectionID).To(Equal(openBankingConnectionID))
		})

		It("should handle missing PSUID and OpenBankingConnectionID", func() {
			refTime := time.Now().UTC().Truncate(time.Second)

			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-no-psu")
			account.SetName("No PSU Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(2500.0)
			balance.SetIsoCurrencyCode("CAD")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

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
			Expect(psp.Amount).To(Equal(big.NewInt(250000)))

			// Verify PSUID is nil when not provided
			Expect(psp.PsuID).To(BeNil())

			// Verify OpenBankingConnectionID is nil when not provided
			Expect(psp.OpenBankingConnectionID).To(BeNil())
		})

		It("should handle large amounts correctly", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-large")
			account.SetName("Large Amount Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(999999999.99)
			balance.SetIsoCurrencyCode("USD")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw: accountBytes,
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-large"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("USD/2"))
			// 999999999.99 * 100 = 99999999999
			expectedAmount := big.NewInt(99999999999)
			Expect(psp.Amount).To(Equal(expectedAmount))
		})

		It("should handle decimal precision correctly", func() {
			refTime := time.Now().UTC().Truncate(time.Second)
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-precision")
			account.SetName("Precision Test Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(123.45)
			balance.SetIsoCurrencyCode("USD")
			balance.SetLastUpdatedDatetime(refTime)
			account.SetBalances(*balance)

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw: accountBytes,
			}

			psp, err := toPSPBalance(pspAccount)
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-precision"))
			Expect(psp.CreatedAt.UTC()).To(Equal(refTime))
			Expect(psp.Asset).To(Equal("USD/2"))
			// 123.45 * 100 = 12345
			expectedAmount := big.NewInt(12345)
			Expect(psp.Amount).To(Equal(expectedAmount))
		})

		It("should use current date when lastUpdatedDatetime is empty", func() {
			account := plaid.NewAccountBaseWithDefaults()
			account.SetAccountId("acc-empty-date")
			account.SetName("Empty Date Account")
			account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

			balance := plaid.NewAccountBalanceWithDefaults()
			balance.SetCurrent(100.0)
			balance.SetIsoCurrencyCode("USD")
			// Don't set LastUpdatedDatetime - it should be empty/zero
			account.SetBalances(*balance)

			accountBytes, err := json.Marshal(account)
			Expect(err).To(BeNil())

			pspAccount := connector.PSPAccount{
				Raw: accountBytes,
			}

			beforeTest := time.Now().UTC()
			psp, err := toPSPBalance(pspAccount)
			afterTest := time.Now().UTC()
			Expect(err).To(BeNil())
			Expect(psp.AccountReference).To(Equal("acc-empty-date"))
			Expect(psp.Asset).To(Equal("USD/2"))
			Expect(psp.Amount).To(Equal(big.NewInt(10000))) // 100.0 * 100

			// The CreatedAt should be between beforeTest and afterTest
			Expect(psp.CreatedAt.UTC()).To(BeTemporally(">=", beforeTest))
			Expect(psp.CreatedAt.UTC()).To(BeTemporally("<=", afterTest))
		})
	})
})
