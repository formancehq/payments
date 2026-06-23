package routable

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Routable createPayout / pollPayableStatus", func() {
	var (
		ctrl   *gomock.Controller
		mock   *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mock = client.NewMockClient(ctrl)
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			name:   "routable",
			logger: logger,
			client: mock,
			config: Config{ActingTeamMember: "tm_default"},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	pi := func() models.PSPPaymentInitiation {
		return models.PSPPaymentInitiation{
			Reference:          "pi_1",
			CreatedAt:          time.Now().UTC(),
			Description:        "rent",
			Amount:             big.NewInt(12345), // 123.45 USD
			Asset:              "USD/2",
			SourceAccount:      &models.PSPAccount{Reference: "acc_1"},
			DestinationAccount: &models.PSPAccount{Reference: "co_1"},
		}
	}

	It("returns PollingPayoutID for non-terminal payables", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, int, error) {
			Expect(req.Type).To(Equal(mappers.DefaultPayableType))
			Expect(req.DeliveryMethod).To(Equal(mappers.DefaultDeliveryMethod))
			Expect(req.PayToCompany).To(Equal("co_1"))
			Expect(req.WithdrawFromAccount).To(Equal("acc_1"))
			Expect(req.Amount).To(Equal("123.45"))
			Expect(req.CurrencyCode).To(Equal("USD"))
			Expect(req.ActingTeamMember).To(Equal("tm_default"))
			Expect(req.IdempotencyKey).To(Equal("pi_1"))
			// Routable's v1 schema requires both: line_items[0].description
			// non-empty AND send_on present (null = send-now).
			Expect(req.LineItems).To(HaveLen(1))
			Expect(req.LineItems[0].Description).NotTo(BeEmpty())
			return &client.Payable{ID: "pa_1", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, http.StatusCreated, nil
		})

		resp, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil())
		Expect(resp.Payment).To(BeNil())
		Expect(resp.PollingPayoutID).NotTo(BeNil())
		Expect(*resp.PollingPayoutID).To(Equal("pa_1"))
	})

	It("synthesizes a non-empty line description and emits send_on as JSON null when the PI has neither", func(ctx SpecContext) {
		bare := pi()
		bare.Description = "" // no description from PI
		bare.Metadata = nil   // no metadata override either

		var captured client.CreatePayableRequest
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, int, error) {
			captured = req
			return &client.Payable{ID: "pa_b", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, http.StatusCreated, nil
		})

		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: bare})
		Expect(err).To(BeNil())
		Expect(captured.LineItems[0].Description).NotTo(BeEmpty(), "Routable rejects payables with empty line_items[0].description")
		// SendOn is nil by design; serialization must preserve that as JSON null.
		Expect(captured.SendOn).To(BeNil())
		body, err := json.Marshal(captured)
		Expect(err).To(BeNil())
		Expect(string(body)).To(ContainSubstring(`"send_on":null`))
	})

	It("returns the Payment immediately when the response is terminal", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).Return(
			&client.Payable{ID: "pa_2", Status: "completed", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			http.StatusCreated,
			nil,
		)
		resp, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil())
		Expect(resp.PollingPayoutID).To(BeNil())
		Expect(resp.Payment).NotTo(BeNil())
		Expect(resp.Payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
	})

	// Async 202 path: Routable echoes only {id}. The plugin must return
	// PollingPayoutID without trying to map the half-empty payable
	// (which would error out on the missing currency / amount).
	// Regression for the bug Quentin flagged on the polling design.
	It("returns PollingPayoutID for async 202 responses without attempting to map", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).Return(
			&client.Payable{ID: "pa_async"}, // 202 body: just {id}
			http.StatusAccepted,
			nil,
		)
		resp, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil(), "must NOT error on the missing currency in a 202 body")
		Expect(resp.Payment).To(BeNil())
		Expect(resp.PollingPayoutID).NotTo(BeNil())
		Expect(*resp.PollingPayoutID).To(Equal("pa_async"))
	})

	// Defensive contract: a 2xx non-error response with no ID is a
	// Routable contract violation; surface it as an error rather than
	// returning an empty PollingPayoutID that the engine would dutifully
	// keep polling forever.
	It("rejects an empty payable ID from upstream", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).Return(
			&client.Payable{ID: ""},
			http.StatusAccepted,
			nil,
		)
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("empty payable"))
	})

	It("respects metadata overrides for type, delivery_method, and acting_team_member", func(ctx SpecContext) {
		piWithOverrides := pi()
		piWithOverrides.Metadata = map[string]string{
			mappers.MetadataKeyType:             "wire",
			mappers.MetadataKeyDeliveryMethod:   "wire",
			mappers.MetadataKeyActingTeamMember: "tm_override",
			mappers.MetadataKeyExternalID:       "ext_42",
		}
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, int, error) {
			Expect(req.Type).To(Equal("wire"))
			Expect(req.DeliveryMethod).To(Equal("wire"))
			Expect(req.ActingTeamMember).To(Equal("tm_override"))
			Expect(req.ExternalID).To(Equal("ext_42"))
			return &client.Payable{ID: "pa_3", Status: "processing", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, http.StatusCreated, nil
		})
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: piWithOverrides})
		Expect(err).To(BeNil())
	})

	It("forwards com.routable.spec/message to Routable.message when set", func(ctx SpecContext) {
		piWithMsg := pi()
		piWithMsg.Metadata = map[string]string{
			mappers.MetadataKeyMessage: "Hi Acme - invoice #12345 for May services.",
		}
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, int, error) {
			Expect(req.Message).To(Equal("Hi Acme - invoice #12345 for May services."))
			return &client.Payable{ID: "pa_msg", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, http.StatusCreated, nil
		})
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: piWithMsg})
		Expect(err).To(BeNil())
	})

	It("omits message from the wire body when no metadata is set", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, int, error) {
			Expect(req.Message).To(BeEmpty())
			body, err := json.Marshal(req)
			Expect(err).To(BeNil())
			Expect(string(body)).NotTo(ContainSubstring(`"message"`))
			return &client.Payable{ID: "pa_nomsg", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, http.StatusCreated, nil
		})
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil())
	})

	// Validation errors must wrap ErrInvalidRequest so Temporal stops
	// retrying invalid PIs.
	It("rejects payment initiations with no source/destination and wraps ErrInvalidRequest", func(ctx SpecContext) {
		bad := pi()
		bad.SourceAccount = nil
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: bad})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, models.ErrInvalidRequest)).To(BeTrue(), "missing SourceAccount must wrap ErrInvalidRequest")

		bad = pi()
		bad.DestinationAccount = nil
		_, err = plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: bad})
		Expect(err).To(HaveOccurred(), "missing DestinationAccount must also be rejected before any network call")
		Expect(errors.Is(err, models.ErrInvalidRequest)).To(BeTrue(), "missing DestinationAccount must wrap ErrInvalidRequest")

		bad = pi()
		bad.Reference = ""
		_, err = plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: bad})
		Expect(errors.Is(err, models.ErrInvalidRequest)).To(BeTrue(), "missing Reference must wrap ErrInvalidRequest")

		bad = pi()
		bad.Asset = ""
		_, err = plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: bad})
		Expect(errors.Is(err, models.ErrInvalidRequest)).To(BeTrue(), "missing Asset must wrap ErrInvalidRequest")

		bad = pi()
		bad.Asset = "ZZZ/2"
		_, err = plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: bad})
		Expect(errors.Is(err, models.ErrInvalidRequest)).To(BeTrue(), "unsupported currency must wrap ErrInvalidRequest")
	})

	// Idempotency contract: every POST /v1/payables MUST carry
	// Idempotency-Key = pi.Reference, on BOTH the payout and the transfer
	// surface. Routable's idempotency window (24h) plus Temporal's
	// at-least-once activity execution means a missing or non-stable key
	// causes duplicate payables = duplicate disbursements at scale. Pin
	// the contract narrowly here so a future refactor of initiatePayable
	// trips on this single assertion.
	Describe("idempotency contract", func() {
		It("uses pi.Reference as the Idempotency-Key on createPayout", func(ctx SpecContext) {
			mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, int, error) {
				Expect(req.IdempotencyKey).To(Equal("pi_1"))
				return &client.Payable{ID: "pa_idem", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, http.StatusCreated, nil
			})
			_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
			Expect(err).To(BeNil())
		})

		It("uses pi.Reference as the Idempotency-Key on createTransfer", func(ctx SpecContext) {
			mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, int, error) {
				Expect(req.IdempotencyKey).To(Equal("pi_1"))
				return &client.Payable{ID: "pa_idem", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, http.StatusCreated, nil
			})
			_, err := plg.createTransfer(ctx, models.CreateTransferRequest{PaymentInitiation: pi()})
			Expect(err).To(BeNil())
		})
	})

	It("falls back to the per-request metadata acting_team_member when the config is empty", func(ctx SpecContext) {
		plg.config = Config{} // no connector-level default
		piWithTM := pi()
		piWithTM.Metadata = map[string]string{mappers.MetadataKeyActingTeamMember: "tm_from_metadata"}
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, int, error) {
			Expect(req.ActingTeamMember).To(Equal("tm_from_metadata"))
			return &client.Payable{ID: "pa_md", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, http.StatusCreated, nil
		})
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: piWithTM})
		Expect(err).To(BeNil())
	})

	It("polls and returns the Payment when the payable is terminal", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_1").Return(
			&client.Payable{ID: "pa_1", Status: "completed", Amount: "10.00", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			nil,
		)
		resp, err := plg.pollPayableStatus(ctx, "pa_1")
		Expect(err).To(BeNil())
		Expect(resp.Payment).NotTo(BeNil())
		Expect(resp.Error).To(BeNil())
	})

	// Terminal failures return Payment (not Error) so the engine links
	// PI ↔ Payment regardless of outcome.
	It("returns the Payment (not an Error) for failed/cancelled/expired terminal states", func(ctx SpecContext) {
		for _, tc := range []struct {
			raw    string
			mapped models.PaymentStatus
		}{
			{"failed", models.PAYMENT_STATUS_FAILED},
			{"canceled", models.PAYMENT_STATUS_CANCELLED},
			{"expired", models.PAYMENT_STATUS_EXPIRED},
		} {
			mock.EXPECT().GetPayable(gomock.Any(), "pa_"+tc.raw).Return(
				&client.Payable{ID: "pa_" + tc.raw, Status: tc.raw, Amount: "10.00", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
				nil,
			)
			resp, err := plg.pollPayableStatus(ctx, "pa_"+tc.raw)
			Expect(err).To(BeNil())
			Expect(resp.Error).To(BeNil(), "must NOT surface terminal failure via response.Error (orphans the Payment)")
			Expect(resp.Payment).NotTo(BeNil(), "must return the Payment so engine can link PI ↔ Payment")
			Expect(resp.Payment.Status).To(Equal(tc.mapped))
		}
	})

	// Routable's eventual-consistency window: a 202 from POST /v1/payables
	// can be followed by N seconds of GET /v1/payables/{id} returning 404
	// before the row is fully indexed. The plugin must surface this as
	// "not yet, retry on schedule" — empty response, no error — so the
	// engine's PollPayoutStatus workflow keeps polling under its standard
	// retry policy. Returning an error here would burn the retry budget
	// and falsely fail the payout.
	It("returns an empty response (engine retries later) on 404 ErrPayableNotFound", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_pending").Return(nil, client.ErrPayableNotFound)
		resp, err := plg.pollPayableStatus(ctx, "pa_pending")
		Expect(err).To(BeNil())
		Expect(resp.Payment).To(BeNil())
		Expect(resp.Error).To(BeNil())
	})

	It("keeps returning the empty response across successive 404s during the eventual-consistency window", func(ctx SpecContext) {
		// Three consecutive 404s (Routable's typical post-202 latency
		// window). Each must surface as the same "not yet" empty
		// response so the engine just keeps polling — no accumulating
		// error state, no transition to a terminal failure.
		mock.EXPECT().GetPayable(gomock.Any(), "pa_window").Return(nil, client.ErrPayableNotFound).Times(3)
		for i := 0; i < 3; i++ {
			resp, err := plg.pollPayableStatus(ctx, "pa_window")
			Expect(err).To(BeNil(), "poll %d must NOT return an error", i+1)
			Expect(resp.Payment).To(BeNil())
			Expect(resp.Error).To(BeNil())
		}
	})

	It("propagates other client errors", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_y").Return(nil, errors.New("boom"))
		_, err := plg.pollPayableStatus(ctx, "pa_y")
		Expect(err).To(HaveOccurred())
	})

	// Link PI ↔ Payment as soon as the payable is visible, regardless of status.
	It("returns the Payment immediately when the payable is still pending", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_z").Return(
			&client.Payable{ID: "pa_z", Status: "pending", Amount: "10.00", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			nil,
		)
		resp, err := plg.pollPayableStatus(ctx, "pa_z")
		Expect(err).To(BeNil())
		Expect(resp.Payment).NotTo(BeNil())
		Expect(resp.Payment.Status).To(Equal(models.PAYMENT_STATUS_PENDING))
		Expect(resp.Error).To(BeNil())
	})
})
