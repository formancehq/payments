package client

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Qonto *Plugin Suite")
}

var _ = Describe("qontoErrors Error", func() {
	DescribeTable("Error method",
		func(qontoError qontoErrors, expected string) {
			err := qontoError.Error()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(expected))
		},
		Entry("no errors in list", qontoErrors{StatusCode: 500}, "statusCode=500, errorMessage=\"unexpected error\""),
		Entry("single error in list",
			qontoErrors{
				StatusCode: 400,
				Errors: []qontoError{
					{Code: "invalid_request", Detail: "The request is invalid"},
				},
			}, "statusCode=400, errorCode=\"invalid_request\", errorMessage=\"The request is invalid\""),
		Entry("multiple errors in list",
			qontoErrors{
				StatusCode: 422,
				Errors: []qontoError{
					{Code: "unprocessable_entity", Detail: "Entity cannot be processed"},
					{Code: "duplicate_entry", Detail: "Duplicate entry found"},
				},
			}, "multiple errors (statusCode=422):"+
				" [errorCode=\"unprocessable_entity\", errorMessage=\"Entity cannot be processed\"]"+
				" [errorCode=\"duplicate_entry\", errorMessage=\"Duplicate entry found\"]"),
		Entry("single error with empty code and detail",
			qontoErrors{
				StatusCode: 400,
				Errors: []qontoError{
					{Code: "", Detail: ""},
				},
			}, "statusCode=400, errorCode=\"\", errorMessage=\"\""),
		Entry("single error with empty detail",
			qontoErrors{
				StatusCode: 400,
				Errors: []qontoError{
					{Code: "missing_detail", Detail: ""},
				},
			}, "statusCode=400, errorCode=\"missing_detail\", errorMessage=\"\""),
		Entry("multiple errors with one empty code or detail",
			qontoErrors{
				StatusCode: 422,
				Errors: []qontoError{
					{Code: "unprocessable_entity", Detail: "Entity cannot be processed"},
					{Code: "", Detail: "Missing code"},
					{Code: "duplicate_entry", Detail: ""},
				},
			}, "multiple errors (statusCode=422):"+
				" [errorCode=\"unprocessable_entity\", errorMessage=\"Entity cannot be processed\"]"+
				" [errorCode=\"\", errorMessage=\"Missing code\"]"+
				" [errorCode=\"duplicate_entry\", errorMessage=\"\"]"),
	)
})
