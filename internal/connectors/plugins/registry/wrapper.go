package registry

import (
	"context"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

type impl struct {
	logger logging.Logger
	plugin models.Plugin
}

func New(logger logging.Logger, plugin models.Plugin) *impl {
	return &impl{
		logger: logger,
		plugin: plugin,
	}
}

func (i *impl) Name() string {
	return i.plugin.Name()
}

func (i *impl) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	name := i.plugin.Name()

	ctx, span := otel.StartSpan(ctx, "plugin.Install", attribute.String("psp", name))
	defer span.End()

	i.logger.WithField("name", name).Info("installing...")

	resp, err := i.plugin.Install(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("install failed: %v", err)
		otel.RecordError(span, err)
		return models.InstallResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("installed!")

	return resp, nil
}

func (i *impl) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.Uninstall", attribute.String("psp", name), attribute.String("connector_id", req.ConnectorID))
	defer span.End()

	i.logger.WithField("name", name).Info("uninstalling...")

	resp, err := i.plugin.Uninstall(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("uninstall failed: %v", err)
		otel.RecordError(span, err)
		return models.UninstallResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("uninstalled!")

	return resp, nil
}

func (i *impl) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextAccounts", attribute.String("psp", name))
	defer span.End()

	i.logger.WithField("name", name).Info("fetching next accounts...")

	resp, err := i.plugin.FetchNextAccounts(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("fetching next accounts failed: %v", err)
		otel.RecordError(span, err)
		return models.FetchNextAccountsResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("fetched next accounts succeeded!")

	return resp, nil
}

func (i *impl) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextExternalAccounts", attribute.String("psp", name))
	defer span.End()

	i.logger.WithField("name", name).Info("fetching next external accounts...")

	resp, err := i.plugin.FetchNextExternalAccounts(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("fetching next external accounts failed: %v", err)
		otel.RecordError(span, err)
		return models.FetchNextExternalAccountsResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("fetched next external accounts succeeded!")

	return resp, nil
}

func (i *impl) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextPayments", attribute.String("psp", name))
	defer span.End()

	i.logger.WithField("name", name).Info("fetching next payments...")

	resp, err := i.plugin.FetchNextPayments(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("fetching next payments failed: %v", err)
		otel.RecordError(span, err)
		return models.FetchNextPaymentsResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("fetched next payments succeeded!")

	return resp, nil
}

func (i *impl) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextBalances", attribute.String("psp", name))
	defer span.End()

	i.logger.WithField("name", name).Info("fetching next balances...")

	resp, err := i.plugin.FetchNextBalances(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("fetching next balances failed: %v", err)
		otel.RecordError(span, err)
		return models.FetchNextBalancesResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("fetched next balances succeeded!")

	return resp, nil
}

func (i *impl) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextOthers", attribute.String("psp", name))
	defer span.End()

	i.logger.WithField("name", name).Info("fetching next others...")

	resp, err := i.plugin.FetchNextOthers(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("fetching next others failed: %v", err)
		otel.RecordError(span, err)
		return models.FetchNextOthersResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("fetched next others succeeded!")

	return resp, nil
}

func (i *impl) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.CreateBankAccount", attribute.String("psp", name), attribute.String("bankAccount.id", req.BankAccount.ID.String()))
	defer span.End()

	i.logger.WithField("name", name).Info("creating bank account...")

	resp, err := i.plugin.CreateBankAccount(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("creating bank account failed: %v", err)
		otel.RecordError(span, err)
		return models.CreateBankAccountResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("created bank account succeeded!")

	return resp, nil
}

func (i *impl) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.CreateTransfer", attribute.String("psp", name), attribute.String("reference", req.PaymentInitiation.Reference))
	defer span.End()

	i.logger.WithField("name", name).Info("creating transfer...")

	resp, err := i.plugin.CreateTransfer(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("creating transfer failed: %v", err)
		otel.RecordError(span, err)
		return models.CreateTransferResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("created transfer succeeded!")

	return resp, nil
}

func (i *impl) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.ReverseTransfer", attribute.String("psp", name), attribute.String("reference", req.PaymentInitiationReversal.Reference))
	defer span.End()

	i.logger.WithField("name", name).Info("reversing transfer...")

	resp, err := i.plugin.ReverseTransfer(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("reversing transfer failed: %v", err)
		otel.RecordError(span, err)
		return models.ReverseTransferResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("reversed transfer succeeded!")

	return resp, nil
}

func (i *impl) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.PollTransferStatus", attribute.String("psp", name), attribute.String("transferID", req.TransferID))
	defer span.End()

	i.logger.WithField("name", name).Info("polling transfer status...")

	resp, err := i.plugin.PollTransferStatus(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("polling transfer status failed: %v", err)
		otel.RecordError(span, err)
		return models.PollTransferStatusResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("polled transfer status succeeded!")

	return resp, nil
}

func (i *impl) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.CreatePayout", attribute.String("psp", name), attribute.String("reference", req.PaymentInitiation.Reference))
	defer span.End()

	i.logger.WithField("name", name).Info("creating payout...")

	resp, err := i.plugin.CreatePayout(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("creating payout failed: %v", err)
		otel.RecordError(span, err)
		return models.CreatePayoutResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("created payout succeeded!")

	return resp, nil
}

func (i *impl) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.ReversePayout", attribute.String("psp", name), attribute.String("reference", req.PaymentInitiationReversal.Reference))
	defer span.End()

	i.logger.WithField("name", name).Info("reversing payout...")

	resp, err := i.plugin.ReversePayout(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("reversing payout failed: %v", err)
		otel.RecordError(span, err)
		return models.ReversePayoutResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("reversed payout succeeded!")

	return resp, nil
}

func (i *impl) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.PollPayoutStatus", attribute.String("psp", name), attribute.String("payoutID", req.PayoutID))
	defer span.End()

	i.logger.WithField("name", name).Info("polling payout status...")

	resp, err := i.plugin.PollPayoutStatus(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("polling payout status failed: %v", err)
		otel.RecordError(span, err)
		return models.PollPayoutStatusResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("polled payout status succeeded!")

	return resp, nil
}

func (i *impl) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.CreateWebhooks", attribute.String("psp", name), attribute.String("connectorID", req.ConnectorID))
	defer span.End()

	i.logger.WithField("name", name).Info("creating webhooks...")

	resp, err := i.plugin.CreateWebhooks(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("creating webhooks failed: %v", err)
		otel.RecordError(span, err)
		return models.CreateWebhooksResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("created webhooks succeeded!")

	return resp, nil
}

func (i *impl) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	name := i.plugin.Name()
	ctx, span := otel.StartSpan(ctx, "plugin.TranslateWebhook", attribute.String("psp", name), attribute.String("translateWebhookRequest.name", req.Name))
	defer span.End()

	i.logger.WithField("name", name).Info("translating webhook...")

	resp, err := i.plugin.TranslateWebhook(ctx, req)
	if err != nil {
		i.logger.WithField("name", name).Error("translating webhook failed: %v", err)
		otel.RecordError(span, err)
		return models.TranslateWebhookResponse{}, translateError(err)
	}

	i.logger.WithField("name", name).Info("translated webhook succeeded!")

	return resp, nil
}

var _ models.Plugin = &impl{}
