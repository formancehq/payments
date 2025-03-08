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
			Expect(err.Error()).To(ContainSubstring("required field amount must be provided"))
		})

		It("should return an error when sourceAccount is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount: big.NewInt(100),
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("required field sourceAccount must be provided"))
		})

		It("should return an error when sourceAccount reference is missing", func(ctx SpecContext) {
			req := models.PSPPaymentInitiation{
				Amount:        big.NewInt(100),
				SourceAccount: &models.PSPAccount{},
			}
			err := plg.validatePayoutRequests(req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("required field sourceAccount.reference must be provided"))
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
			Expect(err.Error()).To(ContainSubstring("required field destinationAccount must be provided"))
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
			Expect(err.Error()).To(ContainSubstring("required field destinationAccount.reference must be provided"))
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
			Expect(err.Error()).To(ContainSubstring("required field metadata must be provided"))
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
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("required field metadata field %s must be provided", client.ColumnPayoutTypeMetadataKey)))
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
			Expect(err.Error()).To(ContainSubstring("must be one of: ach, wire, realtime, international-wire"))
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
				Expect(err.Error()).To(ContainSubstring("required field asset must be provided"))
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

})
