package client_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	powensclient "github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
)

var _ = Describe("Transaction JSON (marshal/unmarshal)", func() {
	Context("marshal", func() {
		It("formats date fields (Europe/Paris for some)", func() {
			// Set instants in UTC. Expect Paris-local formatting on marshal for Date and LastUpdate.
			// Pick a summer date (CEST UTC+2) and a winter date (CET UTC+1)
			summerUTC := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
			winterUTC := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)

			tr := powensclient.Transaction{
				ID:         123,
				AccountID:  456,
				Date:       summerUTC,
				DateTime:   winterUTC, // formatted using time.DateTime as-is (string), doc says UTC
				Value:      json.Number("12.34"),
				Type:       "credit",
				LastUpdate: winterUTC, // Paris naive DateTime
			}

			b, err := json.Marshal(tr)
			Expect(err).To(BeNil())

			// Parse back as raw map to check string forms
			var m map[string]any
			Expect(json.Unmarshal(b, &m)).To(Succeed())

			Expect(m["id"]).To(Equal(float64(123)))
			Expect(m["id_account"]).To(Equal(float64(456)))
			Expect(m["type"]).To(Equal("credit"))
			Expect(m["value"]).To(Equal(12.34))

			// Date is formatted without timezone conversion, using DateOnly layout
			Expect(m["date"]).To(Equal("2025-07-01"))

			// DateTime is formatted with time.DateTime layout; documentation suggests UTC
			Expect(m["date_time"]).To(Equal("2025-01-15 07:30:45"))

			// LastUpdate is Paris local naive DateTime
			Expect(m["last_update"]).To(Equal("2025-01-15 08:30:45"))
		})
	})

	Context("unmarshal", func() {
		It("parses Powens strings and converts to UTC instants", func() {
			// Provide JSON as Powens would send
			payload := `{
				"id": 321,
				"id_account": 654,
				"date": "2025-07-01",
				"date_time": "2025-01-15 07:30:45",
				"value": "-42.00",
				"type": "debit",
				"last_update": "2025-01-15 08:30:45"
			}`

			var tr powensclient.Transaction
			Expect(json.Unmarshal([]byte(payload), &tr)).To(Succeed())

			Expect(tr.ID).To(Equal(321))
			Expect(tr.AccountID).To(Equal(654))
			Expect(tr.Value.String()).To(Equal("-42.00"))
			Expect(tr.Type).To(Equal("debit"))

			// Note that we do not do timezone conversion here, doing it would change the day
			expectedDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
			Expect(tr.Date).To(Equal(expectedDate))

			// date_time is documented as UTC, so parse as UTC (no conversion); expect exact instant
			expectedDateTime := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)
			Expect(tr.DateTime).To(Equal(expectedDateTime))

			// last_update is Paris naive -> UTC
			expectedLastUpdate := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)
			Expect(tr.LastUpdate).To(Equal(expectedLastUpdate))
		})
	})

	Context("round trip", func() {
		It("marshal then unmarshal yields identical Transaction", func() {
			in := powensclient.Transaction{
				ID:         999,
				AccountID:  888,
				Date:       time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),  // CET period (UTC+1) midnight Paris corresponds to 23:00 UTC
				DateTime:   time.Date(2025, 3, 15, 11, 5, 6, 0, time.UTC), // treated as UTC string
				Value:      json.Number("0.01"),
				Type:       "credit",
				LastUpdate: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), // CEST period (UTC+2)
			}

			b, err := json.Marshal(in)
			Expect(err).To(BeNil())

			var out powensclient.Transaction
			Expect(json.Unmarshal(b, &out)).To(Succeed())

			// Exact equality should hold on all fields except Date's time component, which is dropped to midnight when marshaling as date-only.
			Expect(out.ID).To(Equal(in.ID))
			Expect(out.AccountID).To(Equal(in.AccountID))
			Expect(out.Date).To(Equal(in.Date))
			Expect(out.DateTime).To(Equal(in.DateTime))
			Expect(out.Value).To(Equal(in.Value))
			Expect(out.Type).To(Equal(in.Type))
			Expect(out.LastUpdate).To(Equal(in.LastUpdate))
		})

		It("unmarshal then marshal preserves the JSON date formats", func() {
			payload := `{
				"id": 10,
				"id_account": 20,
				"date": "2025-01-15",
				"date_time": "2025-01-15 07:30:45",
				"value": "100.00",
				"type": "credit",
				"last_update": "2025-01-15 08:30:45"
			}`

			var tr powensclient.Transaction
			Expect(json.Unmarshal([]byte(payload), &tr)).To(Succeed())
			b, err := json.Marshal(tr)
			Expect(err).To(BeNil())

			// Compare specific fields to ensure formats preserved
			var m map[string]any
			Expect(json.Unmarshal(b, &m)).To(Succeed())
			Expect(m["date"]).To(Equal("2025-01-15"))
			Expect(m["date_time"]).To(Equal("2025-01-15 07:30:45"))
			Expect(m["last_update"]).To(Equal("2025-01-15 08:30:45"))
		})

		It("omits empty optional fields on marshal", func() {
			in := powensclient.Transaction{ID: 1, AccountID: 2, Value: json.Number("0"), Type: "debit"}
			// zero-value time.Time fields in struct may be formatted by helpers; ensure they are truly omitted by setting them to zero and expecting omission only if MarshalJSON respects omitempty when formatted strings are empty.
			b, err := json.Marshal(in)
			Expect(err).To(BeNil())
			var m map[string]any
			Expect(json.Unmarshal(b, &m)).To(Succeed())
			Expect(m).To(Not(HaveKey("date")))
			Expect(m).To(Not(HaveKey("date_time")))
			Expect(m).To(Not(HaveKey("last_update")))
		})
	})
})
