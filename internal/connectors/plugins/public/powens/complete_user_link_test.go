package powens

import (
	"encoding/base64"
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Powens *Plugin Complete User Link", func() {
	Context("complete user link", func() {
		var (
			plg models.Plugin
		)

		BeforeEach(func() {
			plg = &Plugin{
				client: &client.MockClient{},
			}
		})

		It("should return an error - missing related attempt", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("related attempt is required"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - missing state", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.OpenBankingConnectionAttempt{
					ID: uuid.New(),
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing state"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - invalid state format", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.OpenBankingConnectionAttempt{
					ID: uuid.New(),
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						StateQueryParamID: {"invalid-state"},
					},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to decode state"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - state mismatch", func(ctx SpecContext) {
			id := uuid.New()
			callbackState := models.CallbackState{
				Randomized: "random-123",
				AttemptID:  id,
			}
			stateBytes, _ := json.Marshal(callbackState)
			encodedState := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(stateBytes)

			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.OpenBankingConnectionAttempt{
					ID: id,
					State: models.CallbackState{
						Randomized: "different-random",
						AttemptID:  id,
					},
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						StateQueryParamID: {encodedState},
					},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("state mismatch"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should return an error - missing connection IDs and error", func(ctx SpecContext) {
			id := uuid.New()
			callbackState := models.CallbackState{
				Randomized: "random-123",
				AttemptID:  id,
			}
			stateBytes, _ := json.Marshal(callbackState)
			encodedState := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(stateBytes)

			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.OpenBankingConnectionAttempt{
					ID:    id,
					State: callbackState,
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						StateQueryParamID: {encodedState},
					},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing connection IDs or error"))
			Expect(resp).To(Equal(models.CompleteUserLinkResponse{}))
		})

		It("should complete user link successfully with connection IDs", func(ctx SpecContext) {
			id := uuid.New()
			callbackState := models.CallbackState{
				Randomized: "random-123",
				AttemptID:  id,
			}
			stateBytes, _ := json.Marshal(callbackState)
			encodedState := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(stateBytes)

			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.OpenBankingConnectionAttempt{
					ID:    id,
					State: callbackState,
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						StateQueryParamID:         {encodedState},
						ConnectionIDsQueryParamID: {"conn-1", "conn-2"},
					},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Success).ToNot(BeNil())
			Expect(resp.Error).To(BeNil())
			Expect(resp.Success.Connections).To(HaveLen(0))
		})

		It("should complete user link with error", func(ctx SpecContext) {
			id := uuid.New()
			callbackState := models.CallbackState{
				Randomized: "random-123",
				AttemptID:  id,
			}
			stateBytes, _ := json.Marshal(callbackState)
			encodedState := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(stateBytes)

			req := models.CompleteUserLinkRequest{
				RelatedAttempt: &models.OpenBankingConnectionAttempt{
					ID:    id,
					State: callbackState,
				},
				HTTPCallInformation: models.HTTPCallInformation{
					QueryValues: map[string][]string{
						StateQueryParamID: {encodedState},
						ErrorQueryParamID: {"user_cancelled"},
					},
				},
			}

			resp, err := plg.CompleteUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Success).To(BeNil())
			Expect(resp.Error).ToNot(BeNil())
			Expect(resp.Error.Error).To(Equal("user_cancelled"))
		})
	})
})
