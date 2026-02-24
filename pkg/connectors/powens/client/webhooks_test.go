package client_test

import (
    "encoding/json"
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    powensclient "github.com/formancehq/payments/pkg/connectors/powens/client"
)

var _ = Describe("Client webhooks JSON (marshal/unmarshal)", func() {
    Context("ConnectionSyncedConnection JSON", func() {
        It("marshals LastUpdate in Europe/Paris naive DateTime and omits when zero", func() {
            // Non-zero time
            winterUTC := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)
            c := powensclient.ConnectionSyncedConnection{
                ID:           1,
                State:        "",
                ErrorMessage: "",
                LastUpdate:   winterUTC,
                Active:       true,
                Accounts:     []powensclient.BankAccount{},
            }
            b, err := json.Marshal(c)
            Expect(err).To(BeNil())
            var m map[string]any
            Expect(json.Unmarshal(b, &m)).To(Succeed())
            Expect(m["id"]).To(Equal(float64(1)))
            Expect(m["active"]).To(Equal(true))
            // Paris is CET on 2025-01-15 -> +1 hour
            Expect(m["last_update"]).To(Equal("2025-01-15 08:30:45"))

            // Zero time should omit
            c.LastUpdate = time.Time{}
            b, err = json.Marshal(c)
            Expect(err).To(BeNil())
            m = map[string]any{}
            Expect(json.Unmarshal(b, &m)).To(Succeed())
            Expect(m).To(Not(HaveKey("last_update")))
        })

        It("unmarshals LastUpdate from Powens string to UTC and leaves zero when missing", func() {
            payload := `{"id":10,"state":"","error_message":"","last_update":"2025-01-15 08:30:45","active":true,"accounts":[]}`
            var c powensclient.ConnectionSyncedConnection
            Expect(json.Unmarshal([]byte(payload), &c)).To(Succeed())
            Expect(c.ID).To(Equal(10))
            Expect(c.Active).To(BeTrue())
            // 08:30 Paris CET -> 07:30 UTC
            expected := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)
            Expect(c.LastUpdate).To(Equal(expected))

            // Missing field
            payload2 := `{"id":10,"state":"","error_message":"","active":true,"accounts":[]}`
            var c2 powensclient.ConnectionSyncedConnection
            Expect(json.Unmarshal([]byte(payload2), &c2)).To(Succeed())
            Expect(c2.LastUpdate.IsZero()).To(BeTrue())
        })

        It("returns error on invalid last_update format", func() {
            payload := `{"id":1,"state":"","error_message":"","last_update":"invalid","active":false,"accounts":[]}`
            var c powensclient.ConnectionSyncedConnection
            err := json.Unmarshal([]byte(payload), &c)
            Expect(err).ToNot(BeNil())
        })
    })

    Context("ConnectionSyncedWebhook JSON", func() {
        It("round trips marshal/unmarshal", func() {
            ba := powensclient.BankAccount{ID: 100, UserID: 1, ConnectionID: 2, Currency: powensclient.Currency{ID: "EUR", Precision: 2}}
            in := powensclient.ConnectionSyncedWebhook{
                User:       powensclient.ConnectionSyncedUser{ID: 42},
                Connection: powensclient.ConnectionSyncedConnection{ID: 2, State: "", Active: true, Accounts: []powensclient.BankAccount{ba}},
            }
            b, err := json.Marshal(in)
            Expect(err).To(BeNil())
            var out powensclient.ConnectionSyncedWebhook
            Expect(json.Unmarshal(b, &out)).To(Succeed())
            Expect(out.User.ID).To(Equal(in.User.ID))
            Expect(out.Connection.ID).To(Equal(in.Connection.ID))
            Expect(out.Connection.Active).To(Equal(in.Connection.Active))
            Expect(len(out.Connection.Accounts)).To(Equal(1))
            Expect(out.Connection.Accounts[0].ID).To(Equal(100))
        })
    })
})
