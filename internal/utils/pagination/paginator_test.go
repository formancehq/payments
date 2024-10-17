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
		It("detects that the total is < pageSize and batch is full", func(_ SpecContext) {
			pageSize := 10
			total := make([]GenericContainer, 0, pageSize)
			batch := make([]GenericContainer, 0, pageSize)

			for i := 0; i < pageSize; i++ {
				if i <= 4 {
					total = append(total, GenericContainer{i})
				}

				batch = append(batch, GenericContainer{i})
			}

			needsMore, hasMore := pagination.ShouldFetchMore(total, batch, pageSize)
			Expect(needsMore).To(BeTrue())
			Expect(hasMore).To(BeTrue())
		})

		It("detects that the total is < pageSize and batch is not full", func(_ SpecContext) {
			pageSize := 10
			total := make([]GenericContainer, 0, pageSize)
			batch := make([]GenericContainer, 0, pageSize)

			for i := 0; i < pageSize; i++ {
				if i <= 4 {
					total = append(total, GenericContainer{i})
					batch = append(batch, GenericContainer{i})
				}
			}

			needsMore, hasMore := pagination.ShouldFetchMore(total, batch, pageSize)
			Expect(needsMore).To(BeTrue())
			Expect(hasMore).To(BeFalse())
		})

		It("detects that the total is == pageSize and batch is full", func(_ SpecContext) {
			pageSize := 10
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

		It("detects that the total is == pageSize and batch is not full", func(_ SpecContext) {
			pageSize := 10
			total := make([]GenericContainer, 0, pageSize)
			batch := make([]GenericContainer, 0, pageSize)

			for i := 0; i < pageSize; i++ {
				if i <= 4 {
					batch = append(batch, GenericContainer{i})
				}
				total = append(total, GenericContainer{i})
			}

			needsMore, hasMore := pagination.ShouldFetchMore(total, batch, pageSize)
			Expect(needsMore).To(BeFalse())
			Expect(hasMore).To(BeFalse())
		})

		It("detects that the total is > pageSize and batch is not full", func(_ SpecContext) {
			pageSize := 10
			total := make([]GenericContainer, 0, pageSize)
			batch := make([]GenericContainer, 0, pageSize)

			for i := 0; i < pageSize+10; i++ {
				if i <= 4 {
					batch = append(batch, GenericContainer{i})
				}
				total = append(total, GenericContainer{i})
			}

			needsMore, hasMore := pagination.ShouldFetchMore(total, batch, pageSize)
			Expect(needsMore).To(BeFalse())
			Expect(hasMore).To(BeTrue())
		})

		It("detects that the total is > pageSize and batch is full", func(_ SpecContext) {
			pageSize := 10
			total := make([]GenericContainer, 0, pageSize)
			batch := make([]GenericContainer, 0, pageSize)

			for i := 0; i < pageSize+10; i++ {
				if i < 10 {
					batch = append(batch, GenericContainer{i})
				}
				total = append(total, GenericContainer{i})
			}

			needsMore, hasMore := pagination.ShouldFetchMore(total, batch, pageSize)
			Expect(needsMore).To(BeFalse())
			Expect(hasMore).To(BeTrue())
		})
	})
})
