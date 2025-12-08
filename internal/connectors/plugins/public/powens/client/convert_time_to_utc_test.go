package client_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	powensclient "github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
)

var _ = Describe("ConvertPowensTimeToUTC", func() {
	Context("valid Paris local time inputs", func() {
		It("converts a Paris midnight time to correct UTC considering DST (summer time)", func() {
			// July 1st is in CEST (UTC+2)
			in := "2025-07-01 00:00:00"
			format := "2006-01-02 15:04:05"
			res, err := powensclient.ConvertPowensTimeToUTC(in, format)
			Expect(err).To(BeNil())

			// 00:00 in Paris (UTC+2) -> 22:00 previous day in UTC
			expected := time.Date(2025, 6, 30, 22, 0, 0, 0, time.UTC)
			Expect(res).To(Equal(expected))
		})

		It("converts a Paris time during winter (CET UTC+1) correctly", func() {
			in := "2025-01-15 08:30:45"
			format := "2006-01-02 15:04:05"
			res, err := powensclient.ConvertPowensTimeToUTC(in, format)
			Expect(err).To(BeNil())

			// 08:30:45 in Paris (UTC+1) -> 07:30:45 UTC
			expected := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)
			Expect(res).To(Equal(expected))
		})

		It("parses custom format and converts to UTC", func() {
			in := "15/03/2025 12:00"
			format := "02/01/2006 15:04"
			res, err := powensclient.ConvertPowensTimeToUTC(in, format)
			Expect(err).To(BeNil())

			// March 15th 2025 in Paris is CET (UTC+1) until last Sunday of March
			expected := time.Date(2025, 3, 15, 11, 0, 0, 0, time.UTC)
			Expect(res).To(Equal(expected))
		})
	})

	Context("error cases", func() {
		It("returns error on invalid input that does not match format", func() {
			in := "not-a-date"
			format := "2006-01-02 15:04:05"
			_, err := powensclient.ConvertPowensTimeToUTC(in, format)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("ConvertUTCToPowensTime", func() {
	Context("UTC to Paris formatted string", func() {
		It("formats UTC time to Paris local naive string during summer time (CEST UTC+2)", func() {
			// 2025-07-01 00:00:00 UTC should be 2025-07-01 02:00:00 in Paris
			in := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
			out, err := powensclient.ConvertUTCToPowensTime(in, time.DateTime)
			Expect(err).To(BeNil())
			Expect(out).To(Equal("2025-07-01 02:00:00"))
		})

		It("formats UTC time to Paris local naive string during winter time (CET UTC+1)", func() {
			// 2025-01-15 07:30:45 UTC should be 2025-01-15 08:30:45 in Paris
			in := time.Date(2025, 1, 15, 7, 30, 45, 0, time.UTC)
			out, err := powensclient.ConvertUTCToPowensTime(in, time.DateTime)
			Expect(err).To(BeNil())
			Expect(out).To(Equal("2025-01-15 08:30:45"))
		})

		It("keeps layout without timezone offset", func() {
			// Ensure no offset or timezone info is present in the output
			in := time.Date(2025, 3, 15, 11, 0, 0, 0, time.UTC)
			out, err := powensclient.ConvertUTCToPowensTime(in, time.DateTime)
			Expect(err).To(BeNil())
			// Attempt to parse output using the naive layout in Europe/Paris to ensure it's valid
			loc, err := time.LoadLocation("Europe/Paris")
			Expect(err).To(BeNil())
			parsed, err := time.ParseInLocation("2006-01-02 15:04:05", out, loc)
			Expect(err).To(BeNil())
			// And when converted back to UTC it should match the original instant in UTC+1 (since 2025-03-15 is CET)
			Expect(parsed.UTC()).To(Equal(in))
		})
	})
})

var _ = Describe("powens time function used in conjunction", func() {
	It("ConvertUTCToPowensTime then ConvertPowensTimeToUTC should be the same", func() {
		in := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
		out, err := powensclient.ConvertUTCToPowensTime(in, time.DateTime)
		Expect(err).To(BeNil())
		out2, err := powensclient.ConvertPowensTimeToUTC(out, "2006-01-02 15:04:05")
		Expect(err).To(BeNil())
		Expect(out2).To(Equal(in))
	})

	It("ConvertPowensTimeToUTC then ConvertUTCToPowensTime should be the same", func() {
		in := "2025-07-01 02:00:00"
		out, err := powensclient.ConvertPowensTimeToUTC(in, "2006-01-02 15:04:05")
		Expect(err).To(BeNil())
		out2, err := powensclient.ConvertUTCToPowensTime(out, time.DateTime)
		Expect(err).To(BeNil())
		Expect(out2).To(Equal(in))
	})
})
