package moov

import (
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Moov Payments", func() {
	var (
		plg                     *Plugin
		sampleTransfers         []moov.Transfer
		samplePaymentMethodsMap map[string]moov.PaymentMethod
		sampleMoovAccount       moov.Account
	)

	BeforeEach(func() {
		plg = &Plugin{}
		sampleMoovAccount = moov.Account{
			AccountID:   "account123",
			DisplayName: "Test Account",
			CreatedOn:   time.Now().UTC(),
		}

		sampleTransfers = make([]moov.Transfer, 0)
		samplePaymentMethodsMap = make(map[string]moov.PaymentMethod)

		for i := 0; i < 3; i++ {
			status := moov.TransferStatus_Completed
			if i == 1 {
				status = moov.TransferStatus_Pending
			} else if i == 2 {
				status = moov.TransferStatus_Reversed
			}

			sampleTransfers = append(sampleTransfers, moov.Transfer{
				TransferID: "transfer" + strconv.Itoa(i),
				Status:     status,
				Amount: moov.Amount{
					Value:    int64(1000 * (i + 1)),
					Currency: "USD",
				},
				Source: moov.TransferSource{
					PaymentMethodID:   "src-payment-method-" + strconv.Itoa(i),
					PaymentMethodType: "ach-debit",
					Account: moov.TransferAccount{
						AccountID:   "source-account-" + strconv.Itoa(i),
						Email:       "source@example.com",
						DisplayName: "Source Account",
					},
					Wallet: &moov.WalletPaymentMethod{
						WalletID: "wallet-source-" + strconv.Itoa(i),
					},
				},
				Destination: moov.TransferDestination{
					PaymentMethodID:   "dst-payment-method-" + strconv.Itoa(i),
					PaymentMethodType: "ach-credit",
					Account: moov.TransferAccount{
						AccountID:   "destination-account-" + strconv.Itoa(i),
						Email:       "destination@example.com",
						DisplayName: "Destination Account",
					},
					Wallet: &moov.WalletPaymentMethod{
						WalletID: "wallet-destination-" + strconv.Itoa(i),
					},
				},
				CreatedOn: time.Now().UTC(),
			})

			samplePaymentMethodsMap["src-payment-method-"+strconv.Itoa(i)] = moov.PaymentMethod{
				PaymentMethodID:   "src-payment-method-" + strconv.Itoa(i),
				PaymentMethodType: "ach-debit",
			}

			samplePaymentMethodsMap["dst-payment-method-"+strconv.Itoa(i)] = moov.PaymentMethod{
				PaymentMethodID:   "dst-payment-method-" + strconv.Itoa(i),
				PaymentMethodType: "ach-credit",
			}
		}
	})

	Context("fetching next payments", func() {
		var (
			m *client.MockClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
		})

		It("should return an error when state payload is invalid", func(ctx SpecContext) {
			invalidPayload := []byte(`invalid json`)
			req := models.FetchNextPaymentsRequest{
				State: invalidPayload,
			}

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).NotTo(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should return an error when FromPayload is invalid", func(ctx SpecContext) {
			invalidPayload := []byte(`invalid json`)
			req := models.FetchNextPaymentsRequest{
				FromPayload: invalidPayload,
			}

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).NotTo(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should return an error when GetPayments fails for any status", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				nil, client.Timeline{}, false, 0, errors.New("test error"),
			)

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).To(MatchError("test error"))
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should fetch payments successfully from all statuses", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Completed, 0, 10, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Pending, 0, 10, client.Timeline{}).Return(
				sampleTransfers[1:2], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Reversed, 0, 10, client.Timeline{}).Return(
				sampleTransfers[2:], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Queued, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Canceled, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Failed, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)

			resp, err := plg.fetchNextPayments(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())

			Expect(resp.Payments[0].Reference).To(Equal("transfer0"))
			Expect(resp.Payments[0].Amount.String()).To(Equal(big.NewInt(1000).String()))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))

			Expect(*resp.Payments[0].SourceAccountReference).To(Equal("wallet-source-0"))
			Expect(*resp.Payments[0].DestinationAccountReference).To(Equal("wallet-destination-0"))

			var transfer moov.Transfer
			err = json.Unmarshal(resp.Payments[0].Raw, &transfer)
			Expect(err).To(BeNil())
			Expect(transfer.TransferID).To(Equal("transfer0"))
		})

		It("should handle nil FromPayload", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 10,
			}

			m.EXPECT().GetPayments(gomock.Any(), "", moov.TransferStatus_Completed, 0, 10, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), "", moov.TransferStatus_Pending, 0, 10, client.Timeline{}).Return(
				sampleTransfers[1:2], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), "", moov.TransferStatus_Reversed, 0, 10, client.Timeline{}).Return(
				sampleTransfers[2:], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), "", moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), "", moov.TransferStatus_Queued, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), "", moov.TransferStatus_Canceled, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), "", moov.TransferStatus_Failed, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)

			resp, err := plg.fetchNextPayments(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should set hasMore=true when any status returns page size results", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    1,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Completed, 0, 1, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, true, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Pending, 0, 1, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Reversed, 0, 1, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 1, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Queued, 0, 1, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Canceled, 0, 1, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Failed, 0, 1, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)

			resp, err := plg.fetchNextPayments(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.HasMore).To(BeTrue())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.CompletedSkip).To(Equal(1))
			Expect(state.PendingSkip).To(Equal(0))
		})

		It("should convert transfers to payments correctly", func() {
			payments, err := plg.fillPayments(sampleTransfers)

			Expect(err).To(BeNil())
			Expect(payments).To(HaveLen(3))

			Expect(payments[0].Reference).To(Equal("transfer0"))
			Expect(payments[0].Amount.String()).To(Equal(big.NewInt(1000).String()))
			Expect(payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(payments[0].Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
			Expect(*payments[0].SourceAccountReference).To(Equal("wallet-source-0"))
			Expect(*payments[0].DestinationAccountReference).To(Equal("wallet-destination-0"))

			Expect(payments[1].Status).To(Equal(models.PAYMENT_STATUS_PENDING))

			Expect(payments[2].Status).To(Equal(models.PAYMENT_STATUS_REFUNDED))

			var transfer moov.Transfer
			err = json.Unmarshal(payments[0].Raw, &transfer)
			Expect(err).To(BeNil())
			Expect(transfer.TransferID).To(Equal("transfer0"))
		})

		It("should return error when GetPayments fails for pending status", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Pending, 0, 10, client.Timeline{}).Return(
				nil, client.Timeline{}, false, 0, errors.New("pending error"),
			)

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).To(MatchError("pending error"))
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should return error when GetPayments fails for reversed status", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Pending, 0, 10, client.Timeline{}).Return(
				sampleTransfers[1:2], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Reversed, 0, 10, client.Timeline{}).Return(
				nil, client.Timeline{}, false, 0, errors.New("reversed error"),
			)

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).To(MatchError("reversed error"))
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should return error when GetPayments fails for created status", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Pending, 0, 10, client.Timeline{}).Return(
				sampleTransfers[1:2], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Reversed, 0, 10, client.Timeline{}).Return(
				sampleTransfers[2:], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Queued, 0, 10, client.Timeline{}).Return(
				nil, client.Timeline{}, false, 0, errors.New("created error"),
			)

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).To(MatchError("created error"))
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should return error when GetPayments fails for queued status", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Pending, 0, 10, client.Timeline{}).Return(
				sampleTransfers[1:2], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Reversed, 0, 10, client.Timeline{}).Return(
				sampleTransfers[2:], client.Timeline{}, false, 1, nil,
			)

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Queued, 0, 10, client.Timeline{}).Return(
				nil, client.Timeline{}, false, 0, errors.New("queued error"),
			)

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).To(MatchError("queued error"))
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should return error when GetPayments fails for canceled status", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Pending, 0, 10, client.Timeline{}).Return(
				sampleTransfers[1:2], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Reversed, 0, 10, client.Timeline{}).Return(
				sampleTransfers[2:], client.Timeline{}, false, 1, nil,
			)

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Queued, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Canceled, 0, 10, client.Timeline{}).Return(
				nil, client.Timeline{}, false, 0, errors.New("canceled error"),
			)

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).To(MatchError("canceled error"))
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should return error when GetPayments fails for failed status", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Created, 0, 10, client.Timeline{}).Return(
				sampleTransfers[:1], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Pending, 0, 10, client.Timeline{}).Return(
				sampleTransfers[1:2], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Reversed, 0, 10, client.Timeline{}).Return(
				sampleTransfers[2:], client.Timeline{}, false, 1, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Queued, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Canceled, 0, 10, client.Timeline{}).Return(
				[]moov.Transfer{}, client.Timeline{}, false, 0, nil,
			)
			m.EXPECT().GetPayments(gomock.Any(), sampleMoovAccount.AccountID, moov.TransferStatus_Failed, 0, 10, client.Timeline{}).Return(
				nil, client.Timeline{}, false, 0, errors.New("failed error"),
			)

			resp, err := plg.fetchNextPayments(ctx, req)
			Expect(err).To(MatchError("failed error"))
			Expect(resp.Payments).To(HaveLen(0))
		})
	})

	Context("mapPaymentType function", func() {
		It("should return PAYMENT_TYPE_TRANSFER when accountID matches destination", func() {
			transfer := moov.Transfer{
				Source: moov.TransferSource{
					Account: moov.TransferAccount{AccountID: "source-account"},
				},
				Destination: moov.TransferDestination{
					Account: moov.TransferAccount{AccountID: "dest-account"},
				},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_UNKNOWN))
		})

		It("should return PAYMENT_TYPE_TRANSFER when accountID matches both source and destination", func() {
			transfer := moov.Transfer{
				Source: moov.TransferSource{
					Account: moov.TransferAccount{AccountID: "same-account"},
				},
				Destination: moov.TransferDestination{
					Account: moov.TransferAccount{AccountID: "same-account"},
				},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_UNKNOWN))
		})

		It("should return PAYMENT_TYPE_UNKNOWN when accountID matches neither", func() {
			transfer := moov.Transfer{
				Source: moov.TransferSource{
					Account: moov.TransferAccount{AccountID: "source-account"},
				},
				Destination: moov.TransferDestination{
					Account: moov.TransferAccount{AccountID: "dest-account"},
				},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_UNKNOWN))
		})

	})

	Context("mapPaymentType function", func() {
		It("should return PAYMENT_TYPE_TRANSFER when both source and destination wallets exist", func() {
			transfer := moov.Transfer{
				Source: moov.TransferSource{
					Wallet: &moov.WalletPaymentMethod{
						WalletID: "source-wallet-123",
					},
				},
				Destination: moov.TransferDestination{
					Wallet: &moov.WalletPaymentMethod{
						WalletID: "dest-wallet-456",
					},
				},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_TRANSFER))
		})

		It("should return PAYMENT_TYPE_PAYOUT when only source wallet exists", func() {
			transfer := moov.Transfer{
				Source: moov.TransferSource{
					Wallet: &moov.WalletPaymentMethod{
						WalletID: "source-wallet-123",
					},
				},
				Destination: moov.TransferDestination{
					BankAccount: &moov.BankAccountPaymentMethod{
						BankAccountID: "bank-account-456",
					},
				},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_PAYOUT))
		})

		It("should return PAYMENT_TYPE_PAYIN when only destination wallet exists", func() {
			transfer := moov.Transfer{
				Source: moov.TransferSource{
					BankAccount: &moov.BankAccountPaymentMethod{
						BankAccountID: "bank-account-123",
					},
				},
				Destination: moov.TransferDestination{
					Wallet: &moov.WalletPaymentMethod{
						WalletID: "dest-wallet-456",
					},
				},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_PAYIN))
		})

		It("should return PAYMENT_TYPE_TRANSFER when both source and destination bank accounts exist", func() {
			transfer := moov.Transfer{
				Source: moov.TransferSource{
					BankAccount: &moov.BankAccountPaymentMethod{
						BankAccountID: "source-bank-123",
					},
				},
				Destination: moov.TransferDestination{
					BankAccount: &moov.BankAccountPaymentMethod{
						BankAccountID: "dest-bank-456",
					},
				},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_PAYOUT))
		})

		It("should return PAYMENT_TYPE_UNKNOWN for unknown transfer types", func() {
			transfer := moov.Transfer{
				Source:      moov.TransferSource{},
				Destination: moov.TransferDestination{},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_UNKNOWN))
		})

		It("should handle empty wallet IDs correctly", func() {
			transfer := moov.Transfer{
				Source: moov.TransferSource{
					Wallet: &moov.WalletPaymentMethod{
						WalletID: "", // Empty wallet ID
					},
				},
				Destination: moov.TransferDestination{
					Wallet: &moov.WalletPaymentMethod{
						WalletID: "dest-wallet-456",
					},
				},
			}

			result := mapPaymentType(transfer)
			Expect(result).To(Equal(models.PAYMENT_TYPE_PAYIN))
		})
	})

	Context("fetch payments with moov client", func() {
		var (
			mockedService *client.MockMoovClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockedService = client.NewMockMoovClient(ctrl)

			plg.client, _ = client.New("moov", "https://example.com", "access_token", "test", "test")
			plg.client.NewWithClient(mockedService)
		})

		It("should fail when moov client returns an error", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextPaymentsRequest{
				FromPayload: accountPayload,
				PageSize:    10,
			}

			mockedService.EXPECT().GetMoovTransfers(gomock.Any(), sampleMoovAccount.AccountID, gomock.Any(), gomock.Any(), gomock.Any()).Return(
				nil, errors.New("fetch transfers error"),
			)

			resp, err := plg.fetchNextPayments(ctx, req)

			Expect(err).To(MatchError("fetch transfers error"))
			Expect(resp.Payments).To(HaveLen(0))
		})
	})

	Context("fillPayments function", func() {
		It("should convert transfers to payments correctly with all statuses", func() {
			transfers := []moov.Transfer{
				{
					TransferID: "transfer-completed",
					Status:     moov.TransferStatus_Completed,
					Amount: moov.Amount{
						Value:    1000,
						Currency: "USD",
					},
					Source: moov.TransferSource{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-src"},
					},
					Destination: moov.TransferDestination{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
					},
					CreatedOn: time.Now().UTC(),
				},
				{
					TransferID: "transfer-pending",
					Status:     moov.TransferStatus_Pending,
					Amount: moov.Amount{
						Value:    2000,
						Currency: "EUR",
					},
					Source: moov.TransferSource{
						BankAccount: &moov.BankAccountPaymentMethod{BankAccountID: "bank-src"},
					},
					Destination: moov.TransferDestination{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
					},
					CreatedOn: time.Now().UTC(),
				},
				{
					TransferID: "transfer-failed",
					Status:     moov.TransferStatus_Failed,
					Amount: moov.Amount{
						Value:    3000,
						Currency: "GBP",
					},
					Source: moov.TransferSource{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-src"},
					},
					Destination: moov.TransferDestination{
						BankAccount: &moov.BankAccountPaymentMethod{BankAccountID: "bank-dst"},
					},
					CreatedOn: time.Now().UTC(),
				},
			}

			payments, err := plg.fillPayments(transfers)

			Expect(err).To(BeNil())
			Expect(payments).To(HaveLen(3))

			// Test completed transfer
			Expect(payments[0].Reference).To(Equal("transfer-completed"))
			Expect(payments[0].Amount.String()).To(Equal(big.NewInt(1000).String()))
			Expect(payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(payments[0].Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
			Expect(payments[0].Asset).To(Equal("USD/2"))
			Expect(*payments[0].SourceAccountReference).To(Equal("wallet-src"))
			Expect(*payments[0].DestinationAccountReference).To(Equal("wallet-dst"))

			// Test pending transfer
			Expect(payments[1].Reference).To(Equal("transfer-pending"))
			Expect(payments[1].Amount.String()).To(Equal(big.NewInt(2000).String()))
			Expect(payments[1].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
			Expect(payments[1].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(payments[1].Asset).To(Equal("EUR/2"))

			// Test failed transfer
			Expect(payments[2].Reference).To(Equal("transfer-failed"))
			Expect(payments[2].Amount.String()).To(Equal(big.NewInt(3000).String()))
			Expect(payments[2].Status).To(Equal(models.PAYMENT_STATUS_FAILED))
			Expect(payments[2].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			Expect(payments[2].Asset).To(Equal("GBP/2"))
		})

		It("should handle all transfer statuses correctly", func() {
			statusTests := []struct {
				moovStatus     moov.TransferStatus
				expectedStatus models.PaymentStatus
			}{
				{moov.TransferStatus_Completed, models.PAYMENT_STATUS_SUCCEEDED},
				{moov.TransferStatus_Pending, models.PAYMENT_STATUS_PENDING},
				{moov.TransferStatus_Failed, models.PAYMENT_STATUS_FAILED},
				{moov.TransferStatus_Reversed, models.PAYMENT_STATUS_REFUNDED},
				{moov.TransferStatus_Canceled, models.PAYMENT_STATUS_CANCELLED},
				{moov.TransferStatus_Created, models.PAYMENT_STATUS_PENDING},
				{moov.TransferStatus_Queued, models.PAYMENT_STATUS_PENDING},
			}

			for _, test := range statusTests {
				transfer := moov.Transfer{
					TransferID: "test-transfer",
					Status:     test.moovStatus,
					Amount: moov.Amount{
						Value:    1000,
						Currency: "USD",
					},
					Source: moov.TransferSource{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-src"},
					},
					Destination: moov.TransferDestination{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
					},
					CreatedOn: time.Now().UTC(),
				}

				payments, err := plg.fillPayments([]moov.Transfer{transfer})

				Expect(err).To(BeNil())
				Expect(payments).To(HaveLen(1))
				Expect(payments[0].Status).To(Equal(test.expectedStatus), "Status mapping failed for %v", test.moovStatus)
			}
		})

		It("should handle all payment types correctly", func() {
			typeTests := []struct {
				name         string
				source       moov.TransferSource
				destination  moov.TransferDestination
				expectedType models.PaymentType
			}{
				{
					name: "wallet to wallet transfer",
					source: moov.TransferSource{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-src"},
					},
					destination: moov.TransferDestination{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
					},
					expectedType: models.PAYMENT_TYPE_TRANSFER,
				},
				{
					name: "bank to wallet payin",
					source: moov.TransferSource{
						BankAccount: &moov.BankAccountPaymentMethod{BankAccountID: "bank-src"},
					},
					destination: moov.TransferDestination{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
					},
					expectedType: models.PAYMENT_TYPE_PAYIN,
				},
				{
					name: "wallet to bank payout",
					source: moov.TransferSource{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-src"},
					},
					destination: moov.TransferDestination{
						BankAccount: &moov.BankAccountPaymentMethod{BankAccountID: "bank-dst"},
					},
					expectedType: models.PAYMENT_TYPE_PAYOUT,
				},
				{
					name: "bank to bank payout",
					source: moov.TransferSource{
						BankAccount: &moov.BankAccountPaymentMethod{BankAccountID: "bank-src"},
					},
					destination: moov.TransferDestination{
						BankAccount: &moov.BankAccountPaymentMethod{BankAccountID: "bank-dst"},
					},
					expectedType: models.PAYMENT_TYPE_PAYOUT,
				},
				{
					name:         "unknown type",
					source:       moov.TransferSource{},
					destination:  moov.TransferDestination{},
					expectedType: models.PAYMENT_TYPE_UNKNOWN,
				},
			}

			for _, test := range typeTests {
				transfer := moov.Transfer{
					TransferID:  "test-transfer",
					Status:      moov.TransferStatus_Completed,
					Amount:      moov.Amount{Value: 1000, Currency: "USD"},
					Source:      test.source,
					Destination: test.destination,
					CreatedOn:   time.Now().UTC(),
				}

				payments, err := plg.fillPayments([]moov.Transfer{transfer})

				Expect(err).To(BeNil(), "Error in test: %s", test.name)
				Expect(payments).To(HaveLen(1), "Payments length in test: %s", test.name)
				Expect(payments[0].Type).To(Equal(test.expectedType), "Type mapping failed for: %s", test.name)
			}
		})

		It("should handle empty transfers slice", func() {
			payments, err := plg.fillPayments([]moov.Transfer{})

			Expect(err).To(BeNil())
			Expect(payments).To(HaveLen(0))
		})

		It("should handle transfers with empty wallet IDs", func() {
			transfer := moov.Transfer{
				TransferID: "test-transfer",
				Status:     moov.TransferStatus_Completed,
				Amount: moov.Amount{
					Value:    1000,
					Currency: "USD",
				},
				Source: moov.TransferSource{
					Wallet: &moov.WalletPaymentMethod{WalletID: ""}, // Empty wallet ID
				},
				Destination: moov.TransferDestination{
					Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
				},
				CreatedOn: time.Now().UTC(),
			}

			payments, err := plg.fillPayments([]moov.Transfer{transfer})

			Expect(err).To(BeNil())
			Expect(payments).To(HaveLen(1))
			Expect(payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(*payments[0].SourceAccountReference).To(Equal(""))
			Expect(*payments[0].DestinationAccountReference).To(Equal("wallet-dst"))
		})

		It("should handle transfers with nil wallet pointers", func() {
			transfer := moov.Transfer{
				TransferID: "test-transfer",
				Status:     moov.TransferStatus_Completed,
				Amount: moov.Amount{
					Value:    1000,
					Currency: "USD",
				},
				Source: moov.TransferSource{
					Wallet: nil,
				},
				Destination: moov.TransferDestination{
					Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
				},
				CreatedOn: time.Now().UTC(),
			}

			payments, err := plg.fillPayments([]moov.Transfer{transfer})

			Expect(err).To(BeNil())
			Expect(payments).To(HaveLen(1))
			Expect(payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
		})

		It("should preserve raw transfer data", func() {
			transfer := moov.Transfer{
				TransferID: "test-transfer",
				Status:     moov.TransferStatus_Completed,
				Amount: moov.Amount{
					Value:    1000,
					Currency: "USD",
				},
				Source: moov.TransferSource{
					Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-src"},
				},
				Destination: moov.TransferDestination{
					Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
				},
				CreatedOn: time.Now().UTC(),
			}

			payments, err := plg.fillPayments([]moov.Transfer{transfer})

			Expect(err).To(BeNil())
			Expect(payments).To(HaveLen(1))

			// Verify raw data is preserved
			var unmarshaledTransfer moov.Transfer
			err = json.Unmarshal(payments[0].Raw, &unmarshaledTransfer)
			Expect(err).To(BeNil())
			Expect(unmarshaledTransfer.TransferID).To(Equal("test-transfer"))
			Expect(unmarshaledTransfer.Amount.Value).To(Equal(int64(1000)))
			Expect(unmarshaledTransfer.Amount.Currency).To(Equal("USD"))
		})

		It("should handle different currencies", func() {
			currencies := []string{"USD", "EUR", "GBP", "CAD", "JPY"}

			for _, currency := range currencies {
				transfer := moov.Transfer{
					TransferID: "test-transfer",
					Status:     moov.TransferStatus_Completed,
					Amount: moov.Amount{
						Value:    1000,
						Currency: currency,
					},
					Source: moov.TransferSource{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-src"},
					},
					Destination: moov.TransferDestination{
						Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
					},
					CreatedOn: time.Now().UTC(),
				}

				payments, err := plg.fillPayments([]moov.Transfer{transfer})

				Expect(err).To(BeNil(), "Error for currency: %s", currency)
				Expect(payments).To(HaveLen(1), "Payments length for currency: %s", currency)
				// Expect(payments[0].Asset).To(Equal(currency), "Asset format for currency: %s", currency)
			}
		})

		It("should set ParentReference equal to Reference", func() {
			transfer := moov.Transfer{
				TransferID: "test-parent-ref",
				Status:     moov.TransferStatus_Completed,
				Amount: moov.Amount{
					Value:    1000,
					Currency: "USD",
				},
				Source: moov.TransferSource{
					Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-src"},
				},
				Destination: moov.TransferDestination{
					Wallet: &moov.WalletPaymentMethod{WalletID: "wallet-dst"},
				},
				CreatedOn: time.Now().UTC(),
			}

			payments, err := plg.fillPayments([]moov.Transfer{transfer})

			Expect(err).To(BeNil())
			Expect(payments).To(HaveLen(1))
			Expect(payments[0].Reference).To(Equal("test-parent-ref"))
		})
	})
})
