package tink

import (
	"errors"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Create User", func() {
	Context("create user", func() {
		var (
			ctrl *gomock.Controller
			plg  connector.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should create user successfully", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"

			req := connector.CreateUserRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID: userID,
					Address: &connector.Address{
						Country: &country,
					},
					ContactDetails: &connector.ContactDetails{
						Locale: pointer.For("fr_FR"),
					},
				},
			}

		expectedUserID := userID.String()
		expectedMarket := "FR"
		expectedLocale := "fr_FR"

		m.EXPECT().CreateUser(gomock.Any(), expectedUserID, expectedMarket, expectedLocale).Return(
			client.CreateUserResponse{
				UserID: "tink_user_123",
			},
			nil,
		)

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.PSPUserID).ToNot(BeNil())
			Expect(*resp.PSPUserID).To(Equal("tink_user_123"))
		})

		It("should return error when payment service user is nil", func(ctx SpecContext) {
			req := connector.CreateUserRequest{
				PaymentServiceUser: nil,
			}

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("payment service user is required"))
			Expect(resp).To(Equal(connector.CreateUserResponse{}))
		})

		It("should return error when payment service user address is nil", func(ctx SpecContext) {
			userID := uuid.New()

			req := connector.CreateUserRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID:      userID,
					Address: nil,
				},
			}

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("payment service user address is required"))
			Expect(resp).To(Equal(connector.CreateUserResponse{}))
		})

		It("should return error when payment service user address country is nil", func(ctx SpecContext) {
			userID := uuid.New()

			req := connector.CreateUserRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID: userID,
					Address: &connector.Address{
						Country: nil,
					},
				},
			}

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("payment service user address country is required"))
			Expect(resp).To(Equal(connector.CreateUserResponse{}))
		})

		It("should return error when payment service user address country is not supported", func(ctx SpecContext) {
			userID := uuid.New()
			country := "US"

			req := connector.CreateUserRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID: userID,
					Address: &connector.Address{
						Country: &country,
					},
				},
			}

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user country"))
			Expect(resp).To(Equal(connector.CreateUserResponse{}))
		})

		It("should return error when payment service user locale is not supported", func(ctx SpecContext) {
			userID := uuid.New()
			locale := "xx_XX" // Unsupported locale

			req := connector.CreateUserRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID: userID,
					Address: &connector.Address{
						Country: pointer.For("FR"),
					},
					ContactDetails: &connector.ContactDetails{
						Locale: &locale,
					},
				},
			}

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported payment service user locale"))
			Expect(resp).To(Equal(connector.CreateUserResponse{}))
		})

		It("should return error when client create user fails", func(ctx SpecContext) {
			userID := uuid.New()
			country := "FR"

			req := connector.CreateUserRequest{
				PaymentServiceUser: &connector.PSPPaymentServiceUser{
					ID: userID,
					Address: &connector.Address{
						Country: &country,
					},
					ContactDetails: &connector.ContactDetails{
						Locale: pointer.For("fr_FR"),
					},
				},
			}

		expectedUserID := userID.String()
		expectedMarket := "FR"
		expectedLocale := "fr_FR"

		m.EXPECT().CreateUser(gomock.Any(), expectedUserID, expectedMarket, expectedLocale).Return(
			client.CreateUserResponse{},
			errors.New("client error"),
		)

			resp, err := plg.CreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(connector.CreateUserResponse{}))
		})
	})
})
