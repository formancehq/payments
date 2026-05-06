package client

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Routable Client Suite")
}

var _ = Describe("ErrorResponse.Error", func() {
	DescribeTable("formats Routable API errors",
		func(in ErrorResponse, expected string) {
			Expect(in.Error()).To(Equal(expected))
		},
		Entry("empty body", ErrorResponse{}, "routable api error: empty body"),
		Entry("message only", ErrorResponse{Message: "boom"}, "routable api error: boom"),
		Entry("code + message",
			ErrorResponse{Code: "validation", Message: "invalid"},
			"routable api error [validation]: invalid"),
		Entry("with field errors",
			ErrorResponse{
				Message: "invalid request",
				Errors: []FieldError{
					{Field: "amount", Message: "must be positive"},
					{Message: "missing acting_team_member"},
				},
			},
			"routable api error: invalid request; details: amount: must be positive, missing acting_team_member"),
	)
})
