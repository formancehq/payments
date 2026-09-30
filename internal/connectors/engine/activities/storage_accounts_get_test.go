package activities_test

import (
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Storage Accounts Get", func() {
	var (
		act       activities.Activities
		p         *connectors.MockManager
		s         *storage.MockStorage
		evts      *events.Events
		accountID models.AccountID
		logger    = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay     = 50 * time.Millisecond
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		evts = &events.Events{}
		p = connectors.NewMockManager(ctrl)
		s = storage.NewMockStorage(ctrl)
		act = activities.New(logger, nil, s, evts, p, delay, 0)
		accountID = models.AccountID{
			Reference: "test",
			ConnectorID: models.ConnectorID{
				Reference: uuid.New(),
				Provider:  "test",
			},
		}
	})

	Context("storage accounts get", func() {
		It("success", func(ctx SpecContext) {
			account := &models.Account{ID: accountID}
			s.EXPECT().AccountsGet(ctx, accountID).Return(account, nil)
			res, err := act.StorageAccountsGet(ctx, accountID)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(account))
		})

		It("reports which account is missing and marks it non retryable", func(ctx SpecContext) {
			s.EXPECT().AccountsGet(ctx, accountID).Return(nil, storage.ErrNotFound)
			_, err := act.StorageAccountsGet(ctx, accountID)

			var appErr *temporal.ApplicationError
			Expect(err).To(BeAssignableToTypeOf(appErr))
			Expect(err).To(MatchError(ContainSubstring(accountID.String())))
			Expect(err).To(MatchError(ContainSubstring(storage.ErrNotFound.Error())))

			appErr = err.(*temporal.ApplicationError)
			Expect(appErr.Type()).To(Equal(activities.ErrTypeStorageNotFound))
			Expect(appErr.NonRetryable()).To(BeTrue())
		})

		It("keeps the generic storage type for other failures", func(ctx SpecContext) {
			s.EXPECT().AccountsGet(ctx, accountID).Return(nil, storage.ErrValidation)
			_, err := act.StorageAccountsGet(ctx, accountID)

			appErr := err.(*temporal.ApplicationError)
			Expect(appErr.Type()).To(Equal(activities.ErrTypeStorage))
		})
	})
})
