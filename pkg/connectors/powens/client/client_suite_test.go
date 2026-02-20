package client_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConvertTimeToUTC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Powens Client Test Suite")
}
