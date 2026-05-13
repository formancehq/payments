package universal_test

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/google/uuid"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Universal *Plugin — bank accounts + extra create paths", func() {
	var (
		ctrl   *gomock.Controller
		mc     *client.MockClient
		plg    *universal.Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		cfg    = json.RawMessage(`{"endpoint":"https://x","apiKey":"k"}`)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mc = client.NewMockClient(ctrl)
		mc.EXPECT().SetIdempotencyHeader(gomock.Any()).AnyTimes()
		var err error
		plg, err = universal.New("u", logger, cfg)
		Expect(err).To(BeNil())
		universal.InjectClient(plg, mc)
	})

	AfterEach(func() { ctrl.Finish() })

	It("CreateBankAccount happy path", func(ctx SpecContext) {
		universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_CREATE_BANK_ACCOUNT})
		mc.EXPECT().CreateBankAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.BankAccountResponse{
			RelatedAccount: client.Account{Reference: "acct_ext_xxx", CreatedAt: time.Now().UTC()},
		}, nil)
		res, err := plg.CreateBankAccount(ctx, models.CreateBankAccountRequest{
			BankAccount: models.BankAccount{ID: uuid.New(), CreatedAt: time.Now().UTC(), Name: "Treasury"},
		})
		Expect(err).To(BeNil())
		Expect(res.RelatedAccount.Reference).To(Equal("acct_ext_xxx"))
	})

	It("CreateBankAccount blocked by capability guard", func(ctx SpecContext) {
		universal.InjectDeclared(plg, []models.Capability{})
		_, err := plg.CreateBankAccount(ctx, models.CreateBankAccountRequest{
			BankAccount: models.BankAccount{ID: uuid.New()},
		})
		Expect(err).To(MatchError(plugins.ErrNotImplemented))
	})

	It("FetchNextOthers translates opaque payloads", func(ctx SpecContext) {
		universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_OTHERS})
		mc.EXPECT().ListOthers(gomock.Any(), "report", gomock.Any()).Return(&client.OthersPage{
			Items: []client.Other{{ID: "x1", Data: map[string]any{"key": "val"}}},
		}, nil)
		res, err := plg.FetchNextOthers(ctx, models.FetchNextOthersRequest{Name: "report", PageSize: 50})
		Expect(err).To(BeNil())
		Expect(res.Others).To(HaveLen(1))
		Expect(res.Others[0].ID).To(Equal("x1"))
	})

	It("FetchNextExternalAccounts works", func(ctx SpecContext) {
		universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS})
		mc.EXPECT().ListExternalAccounts(gomock.Any(), gomock.Any()).Return(&client.AccountsPage{
			Items: []client.Account{{Reference: "ext_a", CreatedAt: time.Now().UTC()}},
		}, nil)
		res, err := plg.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{PageSize: 10})
		Expect(err).To(BeNil())
		Expect(res.ExternalAccounts).To(HaveLen(1))
	})

	It("FetchNextBalances iterates accounts via fallback list", func(ctx SpecContext) {
		universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_BALANCES})
		now := time.Now().UTC()
		mc.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Return(&client.AccountsPage{
			Items: []client.Account{{Reference: "a1", CreatedAt: now}},
		}, nil)
		mc.EXPECT().GetBalances(gomock.Any(), "a1").Return(&client.BalancesResponse{
			Items: []client.Balance{{AccountReference: "a1", CreatedAt: now, Amount: "1000", Asset: "EUR/2"}},
		}, nil)
		res, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{PageSize: 10})
		Expect(err).To(BeNil())
		Expect(res.Balances).To(HaveLen(1))
	})

	It("ReversePayout returns terminal payment", func(ctx SpecContext) {
		universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_CREATE_PAYOUT})
		mc.EXPECT().ReversePayout(gomock.Any(), "rev-1", "init-1", gomock.Any()).Return(&client.PayoutResponse{
			Payment: &client.Payment{Reference: "p1", CreatedAt: time.Now().UTC(), Type: "PAYOUT", Status: "REFUNDED", Amount: "100", Asset: "EUR/2"},
		}, nil)
		_, err := plg.ReversePayout(ctx, models.ReversePayoutRequest{PaymentInitiationReversal: revFixture()})
		Expect(err).To(BeNil())
	})

	It("PollTransferStatus returns nil when not yet final", func(ctx SpecContext) {
		universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_CREATE_TRANSFER})
		mc.EXPECT().GetTransfer(gomock.Any(), "tx-1").Return(&client.TransferResponse{}, nil)
		res, err := plg.PollTransferStatus(ctx, models.PollTransferStatusRequest{TransferID: "tx-1"})
		Expect(err).To(BeNil())
		Expect(res.Payment).To(BeNil())
	})
})
