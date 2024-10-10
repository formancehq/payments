package pagination_test

import (
	"testing"

	"github.com/formancehq/payments/internal/utils/pagination"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pagination Suite")
}

var _ = Describe("ShouldFetchMore", func() {
	type GenericContainer struct {
		Val int
	}

	Context("pagination", func() {
		It("detects when total is max capacity and batch has fewer than page size", func(_ SpecContext) {
			pageSize := 10
			total := make([]GenericContainer, 0, pageSize)
			batch := []GenericContainer{
				{Val: 11},
			}

			for i := 0; i < pageSize; i++ {
				total = append(total, GenericContainer{i})
			}

			needsMore, hasMore := pagination.ShouldFetchMore(total, batch, pageSize)
			Expect(needsMore).To(BeFalse())
			Expect(hasMore).To(BeFalse())
		})

		It("detects when total is max capacity and batch has max page size", func(_ SpecContext) {
			pageSize := 15
			total := make([]GenericContainer, 0, pageSize)
			batch := make([]GenericContainer, 0, pageSize)

			for i := 0; i < pageSize; i++ {
				total = append(total, GenericContainer{i})
				batch = append(batch, GenericContainer{i})
			}

			needsMore, hasMore := pagination.ShouldFetchMore(total, batch, pageSize)
			Expect(needsMore).To(BeFalse())
			Expect(hasMore).To(BeTrue())
		})

		It("detects when total cannot be acheived with current batch size", func(_ SpecContext) {
			pageSize := 8
			total := make([]GenericContainer, 0, pageSize)
			batch := make([]GenericContainer, 0, pageSize)

			for i := 0; i < pageSize-1; i++ {
				total = append(total, GenericContainer{i})
				batch = append(batch, GenericContainer{i})
			}

			needsMore, hasMore := pagination.ShouldFetchMore(total, batch, pageSize)
			Expect(needsMore).To(BeTrue())
			Expect(hasMore).To(BeFalse())
		})
	})
})
