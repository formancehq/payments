package generic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Generic Plugin Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next payments", func() {
		var (
			samplePayments []genericclient.Transaction
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			samplePayments = make([]genericclient.Transaction, 0)
			for i := 0; i < 50; i++ {
				samplePayments = append(samplePayments, genericclient.Transaction{
					Id:                   fmt.Sprint(i),
					CreatedAt:            now.Add(-time.Duration(50-i) * time.Minute).UTC(),
					UpdatedAt:            now.Add(-time.Duration(50-i) * time.Minute).UTC(),
					Currency:             "EUR/2", // UMN format
					Type:                 genericclient.PAYIN,
					Status:               genericclient.SUCCEEDED,
					Amount:               "1000",
					SourceAccountID:      pointer.For("acc1"),
					DestinationAccountID: pointer.For("acc2"),
					Metadata:             map[string]string{"foo": "bar"},
				})
			}
		})

		It("should return an error - get payments error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().ListTransactions(gomock.Any(), int64(1), int64(60), time.Time{}).Return(
				[]genericclient.Transaction{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch next payments - no state no results", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().ListTransactions(gomock.Any(), int64(1), int64(60), time.Time{}).Return(
				[]genericclient.Transaction{},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastUpdatedAtFrom.IsZero()).To(BeTrue())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().ListTransactions(gomock.Any(), int64(1), int64(60), time.Time{}).Return(
				samplePayments,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastUpdatedAtFrom.UTC()).To(Equal(samplePayments[49].UpdatedAt.UTC()))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().ListTransactions(gomock.Any(), int64(1), int64(40), time.Time{}).Return(
				samplePayments[:40],
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAtFrom.UTC()).To(Equal(samplePayments[39].UpdatedAt.UTC()))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(fmt.Sprintf(`{"lastUpdatedAtFrom": "%s"}`, samplePayments[38].UpdatedAt.Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().ListTransactions(gomock.Any(), int64(1), int64(40), samplePayments[38].UpdatedAt.UTC()).Return(
				samplePayments[:40],
				nil,
			)

			m.EXPECT().ListTransactions(gomock.Any(), int64(2), int64(40), samplePayments[38].UpdatedAt.UTC()).Return(
				samplePayments[40:],
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(11))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastUpdatedAtFrom.UTC()).To(Equal(samplePayments[49].UpdatedAt.UTC()))
		})
	})
})

// Additional unit tests using standard testing package for better coverage

func TestMatchPaymentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    genericclient.TransactionType
		expected models.PaymentType
	}{
		{"PAYIN", genericclient.PAYIN, models.PAYMENT_TYPE_PAYIN},
		{"PAYOUT", genericclient.PAYOUT, models.PAYMENT_TYPE_PAYOUT},
		{"TRANSFER", genericclient.TRANSFER, models.PAYMENT_TYPE_TRANSFER},
		{"Unknown", genericclient.TransactionType("UNKNOWN"), models.PAYMENT_TYPE_OTHER},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := matchPaymentType(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchPaymentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    genericclient.TransactionStatus
		expected models.PaymentStatus
	}{
		{"PENDING", genericclient.PENDING, models.PAYMENT_STATUS_PENDING},
		{"PROCESSING", genericclient.PROCESSING, models.PAYMENT_STATUS_PROCESSING},
		{"SUCCEEDED", genericclient.SUCCEEDED, models.PAYMENT_STATUS_SUCCEEDED},
		{"FAILED", genericclient.FAILED, models.PAYMENT_STATUS_FAILED},
		{"CANCELLED", genericclient.CANCELLED, models.PAYMENT_STATUS_CANCELLED},
		{"EXPIRED", genericclient.EXPIRED, models.PAYMENT_STATUS_EXPIRED},
		{"REFUNDED", genericclient.REFUNDED, models.PAYMENT_STATUS_REFUNDED},
		{"REFUNDED_FAILURE", genericclient.REFUNDED_FAILURE, models.PAYMENT_STATUS_REFUNDED_FAILURE},
		{"REFUND_REVERSED", genericclient.REFUND_REVERSED, models.PAYMENT_STATUS_REFUND_REVERSED},
		{"DISPUTE", genericclient.DISPUTE, models.PAYMENT_STATUS_DISPUTE},
		{"DISPUTE_WON", genericclient.DISPUTE_WON, models.PAYMENT_STATUS_DISPUTE_WON},
		{"DISPUTE_LOST", genericclient.DISPUTE_LOST, models.PAYMENT_STATUS_DISPUTE_LOST},
		{"AUTHORISATION", genericclient.AUTHORISATION, models.PAYMENT_STATUS_AUTHORISATION},
		{"CAPTURE", genericclient.CAPTURE, models.PAYMENT_STATUS_CAPTURE},
		{"CAPTURE_FAILED", genericclient.CAPTURE_FAILED, models.PAYMENT_STATUS_CAPTURE_FAILED},
		{"OTHER", genericclient.OTHER, models.PAYMENT_STATUS_OTHER},
		{"Unknown", genericclient.TransactionStatus("UNKNOWN"), models.PAYMENT_STATUS_OTHER},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := matchPaymentStatus(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFillPayments_WithRelatedTransactionID(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	relatedID := "related_tx_123"
	pagedPayments := []genericclient.Transaction{
		{
			Id:                   "tx_1",
			CreatedAt:            now,
			UpdatedAt:            now.Add(time.Second),
			Currency:             "USD/2", // UMN format
			Type:                 genericclient.TRANSFER,
			Status:               genericclient.SUCCEEDED,
			Amount:               "5000",
			RelatedTransactionID: &relatedID,
		},
	}

	oldState := paymentsState{LastUpdatedAtFrom: now.Add(-time.Hour)}

	payments, updatedAts, err := fillPayments(pagedPayments, nil, nil, oldState)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Len(t, updatedAts, 1)
	// When RelatedTransactionID is set, Reference should use it
	require.Equal(t, relatedID, payments[0].Reference)
}

func TestFillPayments_WithoutSourceOrDestination(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	pagedPayments := []genericclient.Transaction{
		{
			Id:        "tx_no_accounts",
			CreatedAt: now,
			UpdatedAt: now.Add(time.Second),
			Currency:  "EUR/2", // UMN format
			Type:      genericclient.PAYIN,
			Status:    genericclient.PENDING,
			Amount:    "1000",
			// No SourceAccountID or DestinationAccountID
		},
	}

	oldState := paymentsState{LastUpdatedAtFrom: now.Add(-time.Hour)}

	payments, _, err := fillPayments(pagedPayments, nil, nil, oldState)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Nil(t, payments[0].SourceAccountReference)
	require.Nil(t, payments[0].DestinationAccountReference)
}

func TestFillPayments_InvalidAmount(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	pagedPayments := []genericclient.Transaction{
		{
			Id:        "tx_bad_amount",
			CreatedAt: now,
			UpdatedAt: now.Add(time.Second),
			Currency:  "EUR/2", // UMN format
			Type:      genericclient.PAYIN,
			Status:    genericclient.SUCCEEDED,
			Amount:    "not-a-number",
		},
	}

	oldState := paymentsState{LastUpdatedAtFrom: now.Add(-time.Hour)}

	payments, _, err := fillPayments(pagedPayments, nil, nil, oldState)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse amount")
	require.Nil(t, payments)
}

func TestFillPayments_SkipsOldPayments(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	pagedPayments := []genericclient.Transaction{
		{
			Id:        "tx_old",
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour), // Before state's LastUpdatedAtFrom
			Currency:  "EUR/2", // UMN format
			Type:      genericclient.PAYIN,
			Status:    genericclient.SUCCEEDED,
			Amount:    "1000",
		},
		{
			Id:        "tx_new",
			CreatedAt: now,
			UpdatedAt: now, // After state's LastUpdatedAtFrom
			Currency:  "EUR/2", // UMN format
			Type:      genericclient.PAYIN,
			Status:    genericclient.SUCCEEDED,
			Amount:    "2000",
		},
	}

	oldState := paymentsState{LastUpdatedAtFrom: now.Add(-time.Hour)}

	payments, updatedAts, err := fillPayments(pagedPayments, nil, nil, oldState)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Len(t, updatedAts, 1)
	require.Equal(t, "tx_new", payments[0].Reference)
}

func TestFillPayments_AllPaymentTypes(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	pagedPayments := []genericclient.Transaction{
		{
			Id:        "tx_payin",
			CreatedAt: now,
			UpdatedAt: now.Add(time.Second),
			Currency:  "EUR/2", // UMN format
			Type:      genericclient.PAYIN,
			Status:    genericclient.SUCCEEDED,
			Amount:    "1000",
		},
		{
			Id:        "tx_payout",
			CreatedAt: now,
			UpdatedAt: now.Add(2 * time.Second),
			Currency:  "EUR/2", // UMN format
			Type:      genericclient.PAYOUT,
			Status:    genericclient.PENDING,
			Amount:    "2000",
		},
		{
			Id:        "tx_transfer",
			CreatedAt: now,
			UpdatedAt: now.Add(3 * time.Second),
			Currency:  "USD/2", // UMN format
			Type:      genericclient.TRANSFER,
			Status:    genericclient.FAILED,
			Amount:    "3000",
		},
	}

	oldState := paymentsState{}

	payments, _, err := fillPayments(pagedPayments, nil, nil, oldState)
	require.NoError(t, err)
	require.Len(t, payments, 3)

	require.Equal(t, models.PAYMENT_TYPE_PAYIN, payments[0].Type)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, payments[0].Status)
	require.Equal(t, "EUR/2", payments[0].Asset)

	require.Equal(t, models.PAYMENT_TYPE_PAYOUT, payments[1].Type)
	require.Equal(t, models.PAYMENT_STATUS_PENDING, payments[1].Status)

	require.Equal(t, models.PAYMENT_TYPE_TRANSFER, payments[2].Type)
	require.Equal(t, models.PAYMENT_STATUS_FAILED, payments[2].Status)
	require.Equal(t, "USD/2", payments[2].Asset)
}

func TestFillPayments_WithMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	pagedPayments := []genericclient.Transaction{
		{
			Id:        "tx_meta",
			CreatedAt: now,
			UpdatedAt: now.Add(time.Second),
			Currency:  "GBP/2", // UMN format
			Type:      genericclient.PAYIN,
			Status:    genericclient.SUCCEEDED,
			Amount:    "5000",
			Metadata:  map[string]string{"order_id": "123", "customer": "test"},
		},
	}

	oldState := paymentsState{}

	payments, _, err := fillPayments(pagedPayments, nil, nil, oldState)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Equal(t, "123", payments[0].Metadata["order_id"])
	require.Equal(t, "test", payments[0].Metadata["customer"])
}

func TestFillPayments_WithSourceAndDestination(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	src := "src_acc_123"
	dst := "dst_acc_456"
	pagedPayments := []genericclient.Transaction{
		{
			Id:                   "tx_accounts",
			CreatedAt:            now,
			UpdatedAt:            now.Add(time.Second),
			Currency:             "EUR/2", // UMN format
			Type:                 genericclient.TRANSFER,
			Status:               genericclient.SUCCEEDED,
			Amount:               "10000",
			SourceAccountID:      &src,
			DestinationAccountID: &dst,
		},
	}

	oldState := paymentsState{}

	payments, _, err := fillPayments(pagedPayments, nil, nil, oldState)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.NotNil(t, payments[0].SourceAccountReference)
	require.NotNil(t, payments[0].DestinationAccountReference)
	require.Equal(t, src, *payments[0].SourceAccountReference)
	require.Equal(t, dst, *payments[0].DestinationAccountReference)
}

func TestFetchNextPayments_InvalidState(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	req := models.FetchNextPaymentsRequest{
		State:    []byte(`{invalid json}`),
		PageSize: 10,
	}

	resp, err := plugin.fetchNextPayments(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, models.FetchNextPaymentsResponse{}, resp)
}

func TestFetchNextPayments_NilState(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	mockClient.EXPECT().ListTransactions(gomock.Any(), int64(1), int64(10), time.Time{}).Return(
		[]genericclient.Transaction{
			{
				Id:        "tx_1",
				CreatedAt: now,
				UpdatedAt: now.Add(time.Second),
				Currency:  "EUR/2", // UMN format
				Type:      genericclient.PAYIN,
				Status:    genericclient.SUCCEEDED,
				Amount:    "1000",
			},
		},
		nil,
	)

	req := models.FetchNextPaymentsRequest{
		State:    nil,
		PageSize: 10,
	}

	resp, err := plugin.fetchNextPayments(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.Payments, 1)
}

func TestPaymentsState_Marshaling(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	state := paymentsState{LastUpdatedAtFrom: now}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	var decoded paymentsState
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.True(t, state.LastUpdatedAtFrom.Equal(decoded.LastUpdatedAtFrom))
}
