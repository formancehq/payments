package powens

import (
	"errors"
	"time"

	"github.com/formancehq/payments/pkg/connectors/powens/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Powens *Plugin Create User", func() {
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
			createUserResponse := client.CreateUserResponse{
				IdUser:    12345,
				AuthToken: "auth-token-123",
				ExpiresIn: 3600,
			}

			m.EXPECT().CreateUser(gomock.Any()).Return(createUserResponse, nil)

			resp, err := plg.CreateUser(ctx, connector.CreateUserRequest{})
			Expect(err).To(BeNil())
			Expect(resp.PermanentToken).ToNot(BeNil())
			Expect(resp.PermanentToken.Token).To(Equal("auth-token-123"))
			Expect(resp.PSPUserID).ToNot(BeNil())
			Expect(*resp.PSPUserID).To(Equal("12345"))
			Expect(resp.Metadata[ExpiresInMetadataKey]).To(Equal("3600"))
			Expect(resp.PermanentToken.ExpiresAt).To(BeTemporally("~", time.Now().Add(3600*time.Second), 2*time.Second))
		})

		It("should create user successfully with zero expires in", func(ctx SpecContext) {
			createUserResponse := client.CreateUserResponse{
				IdUser:    12345,
				AuthToken: "auth-token-123",
				ExpiresIn: 0,
			}

			m.EXPECT().CreateUser(gomock.Any()).Return(createUserResponse, nil)

			resp, err := plg.CreateUser(ctx, connector.CreateUserRequest{})
			Expect(err).To(BeNil())
			Expect(resp.PermanentToken).ToNot(BeNil())
			Expect(resp.PermanentToken.Token).To(Equal("auth-token-123"))
			Expect(resp.PSPUserID).ToNot(BeNil())
			Expect(*resp.PSPUserID).To(Equal("12345"))
			Expect(resp.Metadata[ExpiresInMetadataKey]).To(Equal("0"))
			Expect(resp.PermanentToken.ExpiresAt).To(Equal(time.Time{}))
		})

		It("should return an error - client create user error", func(ctx SpecContext) {
			m.EXPECT().CreateUser(gomock.Any()).Return(client.CreateUserResponse{}, errors.New("client error"))

			resp, err := plg.CreateUser(ctx, connector.CreateUserRequest{})
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("client error"))
			Expect(resp).To(Equal(connector.CreateUserResponse{}))
		})
	})
})
