package column

import (
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Column Plugin Payouts", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("Perform Payout Requests", func() {
		var (
			mockHTTPClient *client.MockHTTPClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com")
			plg.client.SetHttpClient(mockHTTPClient)
		})

		It("should return an error when amount is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMissingAmount.Error()))
		})

		It("should return an error when sourceAccount is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount: big.NewInt(100),
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMissingSourceAccount.Error()))
		})

		It("should return an error when sourceAccount reference is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount:        big.NewInt(100),
				SourceAccount: &models.PSPAccount{},
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrSourceAccountReferenceRequired.Error()))
		})

		It("should return an error when destinationAccount is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount: big.NewInt(100),
				SourceAccount: &models.PSPAccount{
					Reference: "test-ref",
				},
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMissingDestinationAccount.Error()))
		})

		It("should return an error when destinationAccount reference is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount: big.NewInt(100),
				SourceAccount: &models.PSPAccount{
					Reference: "test-ref",
				},
				DestinationAccount: &models.PSPAccount{},
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMissingDestinationAccountReference.Error()))
		})

		It("should return an error when metadata is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount: big.NewInt(100),
				SourceAccount: &models.PSPAccount{
					Reference: "test-ref",
				},
				DestinationAccount: &models.PSPAccount{
					Reference: "test-ref",
				},
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMissingMetadata.Error()))
		})

		It("should return an error when payout type is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount: big.NewInt(100),
				SourceAccount: &models.PSPAccount{
					Reference: "test-ref",
				},
				DestinationAccount: &models.PSPAccount{
					Reference: "test-ref",
				},
				Metadata: map[string]string{},
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field metadata: required field metadata must be provided"))
		})

		It("should return an error when payout type is invalid", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount: big.NewInt(100),
				SourceAccount: &models.PSPAccount{
					Reference: "test-ref",
				},
				DestinationAccount: &models.PSPAccount{
					Reference: "test-ref",
				},
				Metadata: map[string]string{
					client.ColumnPayoutTypeMetadataKey: "invalid-type",
				},
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrInvalidMetadataPayoutType.Error()))
		})

		Context("Wire/Realtime/International-Wire Payout Validation", func() {

			It("should return an error when asset is missing", func(ctx SpecContext) {
				req := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					SourceAccount: &models.PSPAccount{
						Reference: "test-ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "test-ref",
					},
					Metadata: map[string]string{
						client.ColumnPayoutTypeMetadataKey: "wire",
					},
				}
				err := plg.validatePayoutRequests(req)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(ErrMissingAsset.Error()))
			})

			It("should return an error when parsing invalid createdAt timestamp", func(ctx SpecContext) {

				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey: "wire",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					200,
					nil,
				).SetArg(2, client.WirePayoutResponse{
					CreatedAt:     "invalid-timestamp",
					ID:            "test-id",
					Amount:        100,
					CurrencyCode:  "USD",
					BankAccountID: "test-bank",
					Description:   "test description",
					UpdatedAt:     "2021-01-01T00:00:00Z",
				})

				res, err := plg.CreatePayout(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreatePayoutResponse{}))
				Expect(err.Error()).To(ContainSubstring("parsing time \"invalid-timestamp\" as"))
			})

		})

		Context("HTTP Request Creation Errors", func() {
			BeforeEach(func() {
				ctrl := gomock.NewController(GinkgoT())
				mockHTTPClient = client.NewMockHTTPClient(ctrl)
				plg.client = client.New("test", "aseplye", "https://test.com")
				plg.client.SetHttpClient(mockHTTPClient)
			})

			It("should return an error when creating ACH payout request fails", func(ctx SpecContext) {

				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey:      "ach",
							client.ColumnAmountConditionMetadataKey: "test-condition",
						},
						Description: "test description",
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					500,
					fmt.Errorf("mock request creation error"),
				)

				res, err := plg.CreatePayout(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreatePayoutResponse{}))
				Expect(err.Error()).To(ContainSubstring("mock request creation error"))
			})

			It("should return an error when creating wire payout request fails", func(ctx SpecContext) {
				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey: "wire",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					500,
					fmt.Errorf("mock request creation error"),
				)

				res, err := plg.CreatePayout(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreatePayoutResponse{}))
				Expect(err.Error()).To(ContainSubstring("mock request creation error"))
			})

			It("should return an error when creating international-wire payout request fails", func(ctx SpecContext) {
				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey: "international-wire",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					500,
					fmt.Errorf("mock request creation error"),
				)

				res, err := plg.CreatePayout(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreatePayoutResponse{}))
				Expect(err.Error()).To(ContainSubstring("mock request creation error"))
			})

			It("should return an error when creating realtime payout request fails", func(ctx SpecContext) {
				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey: "realtime",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					500,
					fmt.Errorf("mock request creation error"),
				)

				res, err := plg.CreatePayout(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreatePayoutResponse{}))
				Expect(err.Error()).To(ContainSubstring("mock request creation error"))
			})

			Context("Invalid URL Errors", func() {
				BeforeEach(func() {
					plg.client = client.New("test", "aseplye", "http://invalid:port")
					plg.client.SetHttpClient(mockHTTPClient)
				})

				It("should return an error when ACH payout URL is invalid", func(ctx SpecContext) {

					req := models.CreatePayoutRequest{
						PaymentInitiation: models.PSPPaymentInitiation{
							Amount: big.NewInt(100),
							Asset:  "USD/2",
							SourceAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							DestinationAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							Metadata: map[string]string{
								client.ColumnPayoutTypeMetadataKey:      "ach",
								client.ColumnAmountConditionMetadataKey: "test-condition",
							},
							Description: "test-description",
						},
					}

					res, err := plg.CreatePayout(ctx, req)
					Expect(err).ToNot(BeNil())
					Expect(res).To(Equal(models.CreatePayoutResponse{}))
					Expect(err.Error()).To(ContainSubstring("failed to create request"))
				})

				It("should return an error when wire payout URL is invalid", func(ctx SpecContext) {
					req := models.CreatePayoutRequest{
						PaymentInitiation: models.PSPPaymentInitiation{
							Amount: big.NewInt(100),
							Asset:  "USD/2",
							SourceAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							DestinationAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							Metadata: map[string]string{
								client.ColumnPayoutTypeMetadataKey: "wire",
							},
						},
					}

					res, err := plg.CreatePayout(ctx, req)
					Expect(err).ToNot(BeNil())
					Expect(res).To(Equal(models.CreatePayoutResponse{}))
					Expect(err.Error()).To(ContainSubstring("failed to create request"))
				})

				It("should return an error when international wire payout URL is invalid", func(ctx SpecContext) {
					req := models.CreatePayoutRequest{
						PaymentInitiation: models.PSPPaymentInitiation{
							Amount: big.NewInt(100),
							Asset:  "USD/2",
							SourceAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							DestinationAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							Metadata: map[string]string{
								client.ColumnPayoutTypeMetadataKey: "international-wire",
							},
						},
					}

					res, err := plg.CreatePayout(ctx, req)
					Expect(err).ToNot(BeNil())
					Expect(res).To(Equal(models.CreatePayoutResponse{}))
					Expect(err.Error()).To(ContainSubstring("failed to create request"))
				})

				It("should return an error when realtime payout URL is invalid", func(ctx SpecContext) {
					req := models.CreatePayoutRequest{
						PaymentInitiation: models.PSPPaymentInitiation{
							Amount: big.NewInt(100),
							Asset:  "USD/2",
							SourceAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							DestinationAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							Metadata: map[string]string{
								client.ColumnPayoutTypeMetadataKey: "realtime",
							},
						},
					}

					res, err := plg.CreatePayout(ctx, req)
					Expect(err).ToNot(BeNil())
					Expect(res).To(Equal(models.CreatePayoutResponse{}))
					Expect(err.Error()).To(ContainSubstring("failed to create request"))
				})

				It("should return an error when asset is invalid", func(ctx SpecContext) {
					req := models.CreatePayoutRequest{
						PaymentInitiation: models.PSPPaymentInitiation{
							Amount: big.NewInt(100),
							Asset:  "INVALID/2",
							SourceAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							DestinationAccount: &models.PSPAccount{
								Reference: "test-ref",
							},
							Metadata: map[string]string{
								client.ColumnPayoutTypeMetadataKey: "wire",
							},
						},
					}

					res, err := plg.CreatePayout(ctx, req)
					Expect(err).ToNot(BeNil())
					Expect(res).To(Equal(models.CreatePayoutResponse{}))
					Expect(err.Error()).To(ContainSubstring("failed to get currency and precision from asset"))
				})

			})
		})

		Context("Payout to Payment Transformation", func() {
			It("should create a payment from a wire payout response", func(ctx SpecContext) {

				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey: "wire",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					200,
					nil,
				).SetArg(2, client.WirePayoutResponse{
					CreatedAt:     "2021-01-01T00:00:00Z",
					ID:            "test-id",
					Amount:        100,
					CurrencyCode:  "USD",
					BankAccountID: "test-bank",
					Description:   "test description",
					UpdatedAt:     "2021-01-01T00:00:00Z",
				})

				resp, err := plg.CreatePayout(ctx, req)
				Expect(err).To(BeNil())
				Expect(resp.Payment.Reference).To(Equal("test-id"))
				Expect(resp.Payment.Amount).To(Equal(big.NewInt(100)))
				Expect(resp.Payment.Asset).To(Equal("USD/2"))

			})

			It("should create a payment from an ACH payout response", func(ctx SpecContext) {
				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey:      "ach",
							client.ColumnAmountConditionMetadataKey: "test-condition",
						},
						Description: "test-description",
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					200,
					nil,
				).SetArg(2, client.ACHPayoutResponse{
					ID:            "test-id",
					CreatedAt:     "2021-01-01T00:00:00Z",
					UpdatedAt:     "2021-01-01T00:00:00Z",
					Status:        "completed",
					Amount:        100,
					CurrencyCode:  "USD",
					BankAccountID: "test-bank",
					Description:   "test description",
				})

				resp, err := plg.CreatePayout(ctx, req)
				Expect(err).To(BeNil())
				Expect(resp.Payment.Reference).To(Equal("test-id"))
				Expect(resp.Payment.Amount).To(Equal(big.NewInt(100)))

			})

			It("should create a payment from an international wire payout response", func(ctx SpecContext) {
				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey: "international-wire",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					200,
					nil,
				).SetArg(2, client.InternationalWirePayoutResponse{
					CreatedAt:     "2021-01-01T00:00:00Z",
					ID:            "test-id",
					Amount:        100,
					CurrencyCode:  "USD",
					BankAccountID: "test-bank",
					Description:   "test description",
					UpdatedAt:     "2021-01-01T00:00:00Z",
				})

				resp, err := plg.CreatePayout(ctx, req)
				Expect(err).To(BeNil())
				Expect(resp.Payment.Reference).To(Equal("test-id"))
				Expect(resp.Payment.Amount).To(Equal(big.NewInt(100)))
				Expect(resp.Payment.Asset).To(Equal("USD/2"))
			})

			It("should create a payment from a realtime payout response", func(ctx SpecContext) {
				req := models.CreatePayoutRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						SourceAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "test-ref",
						},
						Metadata: map[string]string{
							client.ColumnPayoutTypeMetadataKey: "realtime",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					200,
					nil,
				).SetArg(2, client.RealtimeTransferResponse{
					ID:            "test-id",
					Amount:        100,
					CurrencyCode:  "USD",
					BankAccountID: "test-bank",
					Description:   "test description",
					InitiatedAt:   "2021-01-01T00:00:00Z",
					IsOnUs:        true,
					IsIncoming:    false,
					Status:        "completed",
				})

				resp, err := plg.CreatePayout(ctx, req)
				Expect(err).To(BeNil())
				Expect(resp.Payment.Reference).To(Equal("test-id"))
				Expect(resp.Payment.Amount).To(Equal(big.NewInt(100)))
				Expect(resp.Payment.Asset).To(Equal("USD/2"))
			})
		})
	})

	Context("mapTransactionStatus", func() {
		It("should map status values correctly", func() {
			testCases := []struct {
				status         string
				expectedStatus models.PaymentStatus
			}{
				{"submitted", models.PAYMENT_STATUS_PENDING},
				{"pending_submission", models.PAYMENT_STATUS_PENDING},
				{"initiated", models.PAYMENT_STATUS_PENDING},
				{"pending_deposit", models.PAYMENT_STATUS_PENDING},
				{"pending_first_return", models.PAYMENT_STATUS_PENDING},
				{"pending_reclear", models.PAYMENT_STATUS_PENDING},
				{"pending_return", models.PAYMENT_STATUS_PENDING},
				{"pending_second_return", models.PAYMENT_STATUS_PENDING},
				{"pending_stop", models.PAYMENT_STATUS_PENDING},
				{"pending_user_initiated_return", models.PAYMENT_STATUS_PENDING},
				{"scheduled", models.PAYMENT_STATUS_PENDING},
				{"pending", models.PAYMENT_STATUS_PENDING},
				{"completed", models.PAYMENT_STATUS_SUCCEEDED},
				{"deposited", models.PAYMENT_STATUS_SUCCEEDED},
				{"recleared", models.PAYMENT_STATUS_SUCCEEDED},
				{"settled", models.PAYMENT_STATUS_SUCCEEDED},
				{"accepted", models.PAYMENT_STATUS_SUCCEEDED},
				{"canceled", models.PAYMENT_STATUS_CANCELLED},
				{"stopped", models.PAYMENT_STATUS_CANCELLED},
				{"blocked", models.PAYMENT_STATUS_CANCELLED},
				{"failed", models.PAYMENT_STATUS_FAILED},
				{"rejected", models.PAYMENT_STATUS_FAILED},
				{"returned", models.PAYMENT_STATUS_REFUNDED},
				{"user_initiated_returned", models.PAYMENT_STATUS_REFUNDED},
				{"return_contested", models.PAYMENT_STATUS_REFUND_REVERSED},
				{"return_dishonored", models.PAYMENT_STATUS_REFUND_REVERSED},
				{"user_initiated_return_dishonored", models.PAYMENT_STATUS_REFUND_REVERSED},
				{"first_return", models.PAYMENT_STATUS_REFUNDED_FAILURE},
				{"second_return", models.PAYMENT_STATUS_REFUNDED_FAILURE},
				{"user_initiated_return_submitted", models.PAYMENT_STATUS_REFUNDED_FAILURE},
				{"manual_review", models.PAYMENT_STATUS_AUTHORISATION},
				{"manual_review_approved", models.PAYMENT_STATUS_AUTHORISATION},
				{"hold", models.PAYMENT_STATUS_CAPTURE},
				{"unknown", models.PAYMENT_STATUS_UNKNOWN},
			}

			for _, tc := range testCases {
				result := plg.mapTransactionStatus(tc.status)
				Expect(result).To(Equal(tc.expectedStatus), "Status %s should map to %s", tc.status, tc.expectedStatus)
			}
		})
	})
})
