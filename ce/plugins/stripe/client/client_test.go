//go:build !contract

// Excluded from contract runs (-tags contract): contract_test.go declares its
// own Ginkgo suite in this package's test binary, and Ginkgo does not support
// two RunSpecs entrypoints in one binary. `just tests` runs with -tags it, so
// this unit suite is unaffected.

package client_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stripe Client Suite")
}
