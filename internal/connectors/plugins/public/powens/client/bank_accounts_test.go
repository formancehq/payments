package client_test

import (
    "encoding/json"
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    powensclient "github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
)

var _ = Describe("BankAccount JSON (marshal/unmarshal)", func() {
    Context("marshal", func() {
        It("formats LastUpdate according to Powens expectations (Europe/Paris naive)", func() {
            // non-zero LastUpdate should be formatted in Paris local using time.DateTime
            winterUTC := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)
            ba := powensclient.BankAccount{
                ID:           10,
                UserID:       20,
                ConnectionID: 30,
                Currency:     powensclient.Currency{ID: "EUR", Name: "Euro", Symbol: "€", Precision: 2},
                OriginalName: "Main",
                Error:        "",
                LastUpdate:   winterUTC,
                Balance:      json.Number("123.45"),
                Transactions: []powensclient.Transaction{{ID: 1, AccountID: 999, Type: "credit", Value: json.Number("1")}},
            }

            b, err := json.Marshal(ba)
            Expect(err).To(BeNil())

            var m map[string]any
            Expect(json.Unmarshal(b, &m)).To(Succeed())

            // Core fields
            Expect(m["id"]).To(Equal(float64(10)))
            Expect(m["id_user"]).To(Equal(float64(20)))
            Expect(m["id_connection"]).To(Equal(float64(30)))
            Expect(m["original_name"]).To(Equal("Main"))
            // Currency remains an object
            Expect(m["currency"]).To(Not(BeNil()))
            // Balance rendered as number
            Expect(m["balance"]).To(Equal(123.45))

            // LastUpdate is Paris local naive DateTime (07:30 UTC -> 08:30 Paris in winter?)
            // On 2025-01-15, Paris is CET (UTC+1)
            Expect(m["last_update"]).To(Equal("2025-01-15 08:30:45"))

            // Transactions array preserved
            Expect(m["transactions"]).ToNot(BeNil())
            txs := m["transactions"].([]any)
            Expect(len(txs)).To(Equal(1))
            tx := txs[0].(map[string]any)
            Expect(tx["id"]).To(Equal(float64(1)))
            Expect(tx["id_account"]).To(Equal(float64(999)))
            Expect(tx["type"]).To(Equal("credit"))
            Expect(tx["value"]).To(Equal(1.0))
        })

        It("omits last_update on zero time", func() {
            ba := powensclient.BankAccount{ID: 1, UserID: 2, ConnectionID: 3}
            b, err := json.Marshal(ba)
            Expect(err).To(BeNil())
            var m map[string]any
            Expect(json.Unmarshal(b, &m)).To(Succeed())
            Expect(m).To(Not(HaveKey("last_update")))
        })
    })

    Context("unmarshal", func() {
        It("parses last_update Powens string into UTC", func() {
            payload := `{
                "id": 100,
                "id_user": 200,
                "id_connection": 300,
                "currency": {"id":"EUR","name":"Euro","symbol":"€","precision":2},
                "original_name": "Savings",
                "error": "",
                "last_update": "2025-01-15 08:30:45",
                "balance": "10.00",
                "transactions": []
            }`

            var ba powensclient.BankAccount
            Expect(json.Unmarshal([]byte(payload), &ba)).To(Succeed())

            Expect(ba.ID).To(Equal(100))
            Expect(ba.UserID).To(Equal(200))
            Expect(ba.ConnectionID).To(Equal(300))
            Expect(ba.Currency.ID).To(Equal("EUR"))
            Expect(ba.OriginalName).To(Equal("Savings"))
            Expect(ba.Error).To(Equal(""))
            Expect(ba.Balance.String()).To(Equal("10.00"))
            Expect(ba.Transactions).To(BeEmpty())

            // Paris 08:30 CET -> 07:30 UTC
            expected := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)
            Expect(ba.LastUpdate).To(Equal(expected))
        })

        It("handles missing last_update as zero time", func() {
            payload := `{"id":1,"id_user":2,"id_connection":3,"currency":{"id":"EUR","name":"Euro","symbol":"€","precision":2},"balance":"0","transactions":[]}`
            var ba powensclient.BankAccount
            Expect(json.Unmarshal([]byte(payload), &ba)).To(Succeed())
            Expect(ba.LastUpdate.IsZero()).To(BeTrue())
        })

        It("returns error on invalid last_update format", func() {
            payload := `{"id":1,"id_user":2,"id_connection":3,"last_update":"invalid","balance":"0","transactions":[]}`
            var ba powensclient.BankAccount
            err := json.Unmarshal([]byte(payload), &ba)
            Expect(err).ToNot(BeNil())
        })
    })

    Context("round trip", func() {
        It("marshal then unmarshal yields identical BankAccount", func() {
            in := powensclient.BankAccount{
                ID:           7,
                UserID:       8,
                ConnectionID: 9,
                Currency:     powensclient.Currency{ID: "USD", Name: "US Dollar", Symbol: "$", Precision: 2},
                OriginalName: "BA",
                Balance:      json.Number("0.01"),
                LastUpdate:   time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), // CEST
            }

            b, err := json.Marshal(in)
            Expect(err).To(BeNil())

            var out powensclient.BankAccount
            Expect(json.Unmarshal(b, &out)).To(Succeed())

            Expect(out.ID).To(Equal(in.ID))
            Expect(out.UserID).To(Equal(in.UserID))
            Expect(out.ConnectionID).To(Equal(in.ConnectionID))
            Expect(out.Currency).To(Equal(in.Currency))
            Expect(out.OriginalName).To(Equal(in.OriginalName))
            Expect(out.Balance).To(Equal(in.Balance))
            Expect(out.LastUpdate).To(Equal(in.LastUpdate))
        })
    })
})
