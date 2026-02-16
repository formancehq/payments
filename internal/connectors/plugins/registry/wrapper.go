package registry

import (
	"context"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

type impl struct {
	connectorID models.ConnectorID
	logger      logging.Logger
	plugin      models.Plugin
}

func New(connectorID models.ConnectorID, logger logging.Logger, plugin models.Plugin) *impl {
	return &impl{
		connectorID: connectorID,
		logger:      logger,
		plugin:      plugin,
	}
}

func (i *impl) Name() string {
	return i.plugin.Name()
}

func (i *impl) IsScheduledForDeletion() bool {
	return i.plugin.IsScheduledForDeletion()
}

func (i *impl) ScheduleForDeletion(isScheduledForDeletion bool) {
	i.plugin.ScheduleForDeletion(isScheduledForDeletion)
}

func (i *impl) Config() models.PluginInternalConfig {
	return i.plugin.Config()
}

func (i *impl) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.Install", attribute.String("psp", i.connectorID.Provider), attribute.String("connector_id", i.connectorID.String()))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("installing...")

	resp, err := i.plugin.Install(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Errorf("install failed: %w", err)
		otel.RecordError(span, err)
		return models.InstallResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("installed!")

	return resp, nil
}

func (i *impl) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.Uninstall", attribute.String("psp", i.connectorID.Provider), attribute.String("connector_id", req.ConnectorID))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("uninstalling...")

	resp, err := i.plugin.Uninstall(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("uninstall failed:", err)
		otel.RecordError(span, err)
		return models.UninstallResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("uninstalled!")

	return resp, nil
}

func (i *impl) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextAccounts", attribute.String("psp", i.connectorID.Provider), attribute.String("connector_id", i.connectorID.String()))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetching next accounts...")

	resp, err := i.plugin.FetchNextAccounts(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("fetching next accounts failed:", err)
		otel.RecordError(span, err)
		return models.FetchNextAccountsResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetched next accounts succeeded!")

	return resp, nil
}

func (i *impl) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextExternalAccounts", attribute.String("psp", i.connectorID.Provider), attribute.String("connector_id", i.connectorID.String()))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetching next external accounts...")

	resp, err := i.plugin.FetchNextExternalAccounts(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("fetching next external accounts failed:", err)
		otel.RecordError(span, err)
		return models.FetchNextExternalAccountsResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetched next external accounts succeeded!")

	return resp, nil
}

func (i *impl) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextPayments", attribute.String("psp", i.connectorID.Provider), attribute.String("connector_id", i.connectorID.String()))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetching next payments...")

	resp, err := i.plugin.FetchNextPayments(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("fetching next payments failed:", err)
		otel.RecordError(span, err)
		return models.FetchNextPaymentsResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetched next payments succeeded!")

	return resp, nil
}

func (i *impl) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextBalances", attribute.String("psp", i.connectorID.Provider), attribute.String("connector_id", i.connectorID.String()))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetching next balances...")

	resp, err := i.plugin.FetchNextBalances(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("fetching next balances failed:", err)
		otel.RecordError(span, err)
		return models.FetchNextBalancesResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetched next balances succeeded!")

	return resp, nil
}

func (i *impl) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.FetchNextOthers", attribute.String("psp", i.connectorID.Provider), attribute.String("connector_id", i.connectorID.String()))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetching next others...")

	resp, err := i.plugin.FetchNextOthers(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("fetching next others failed:", err)
		otel.RecordError(span, err)
		return models.FetchNextOthersResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("fetched next others succeeded!")

	return resp, nil
}

func (i *impl) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.CreateBankAccount", attribute.String("psp", i.connectorID.Provider), attribute.String("bankAccount.id", req.BankAccount.ID.String()))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("creating bank account...")

	resp, err := i.plugin.CreateBankAccount(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("creating bank account failed:", err)
		otel.RecordError(span, err)
		return models.CreateBankAccountResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("created bank account succeeded!")

	return resp, nil
}

func (i *impl) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.CreateTransfer", attribute.String("psp", i.connectorID.Provider), attribute.String("reference", req.PaymentInitiation.Reference))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("creating transfer...")

	resp, err := i.plugin.CreateTransfer(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("creating transfer failed:", err)
		otel.RecordError(span, err)
		return models.CreateTransferResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("created transfer succeeded!")

	return resp, nil
}

func (i *impl) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.ReverseTransfer", attribute.String("psp", i.connectorID.Provider), attribute.String("reference", req.PaymentInitiationReversal.Reference))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("reversing transfer...")

	resp, err := i.plugin.ReverseTransfer(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("reversing transfer failed:", err)
		otel.RecordError(span, err)
		return models.ReverseTransferResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("reversed transfer succeeded!")

	return resp, nil
}

func (i *impl) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.PollTransferStatus", attribute.String("psp", i.connectorID.Provider), attribute.String("transferID", req.TransferID))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("polling transfer status...")

	resp, err := i.plugin.PollTransferStatus(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("polling transfer status failed:", err)
		otel.RecordError(span, err)
		return models.PollTransferStatusResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("polled transfer status succeeded!")

	return resp, nil
}

func (i *impl) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.CreatePayout", attribute.String("psp", i.connectorID.Provider), attribute.String("reference", req.PaymentInitiation.Reference))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("creating payout...")

	resp, err := i.plugin.CreatePayout(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("creating payout failed:", err)
		otel.RecordError(span, err)
		return models.CreatePayoutResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("created payout succeeded!")

	return resp, nil
}

func (i *impl) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.ReversePayout", attribute.String("psp", i.connectorID.Provider), attribute.String("reference", req.PaymentInitiationReversal.Reference))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("reversing payout...")

	resp, err := i.plugin.ReversePayout(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("reversing payout failed:", err)
		otel.RecordError(span, err)
		return models.ReversePayoutResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("reversed payout succeeded!")

	return resp, nil
}

func (i *impl) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.PollPayoutStatus", attribute.String("psp", i.connectorID.Provider), attribute.String("payoutID", req.PayoutID))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("polling payout status...")

	resp, err := i.plugin.PollPayoutStatus(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("polling payout status failed:", err)
		otel.RecordError(span, err)
		return models.PollPayoutStatusResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("polled payout status succeeded!")

	return resp, nil
}

func (i *impl) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.CreateWebhooks", attribute.String("psp", i.connectorID.Provider), attribute.String("connectorID", req.ConnectorID))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("creating webhooks...")

	resp, err := i.plugin.CreateWebhooks(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("creating webhooks failed:", err)
		otel.RecordError(span, err)
		return models.CreateWebhooksResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("created webhooks succeeded!")

	return resp, nil
}

func (i *impl) TrimWebhook(ctx context.Context, req models.TrimWebhookRequest) (models.TrimWebhookResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.TrimWebhook", attribute.String("psp", i.connectorID.Provider), attribute.String("trimWebhookRequest.name", req.Config.Name))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("trimming webhook...")

	resp, err := i.plugin.TrimWebhook(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("trimming webhook failed:", err)
		otel.RecordError(span, err)
		return models.TrimWebhookResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("trimmed webhook succeeded!")

	return resp, nil
}

func (i *impl) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.VerifyWebhook", attribute.String("psp", i.connectorID.Provider), attribute.String("verifyWebhookRequest.name", req.Config.Name))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("verifying webhook...")

	resp, err := i.plugin.VerifyWebhook(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("verifying webhook failed: ", err)
		otel.RecordError(span, err)
		return models.VerifyWebhookResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("verified webhook succeeded!")

	return resp, nil
}

func (i *impl) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.TranslateWebhook", attribute.String("psp", i.connectorID.Provider), attribute.String("translateWebhookRequest.name", req.Name))
	defer span.End()

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("translating webhook...")

	resp, err := i.plugin.TranslateWebhook(ctx, req)
	if err != nil {
		i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Error("translating webhook failed:", err)
		otel.RecordError(span, err)
		return models.TranslateWebhookResponse{}, translateError(err)
	}

	i.logger.WithField("psp", i.connectorID.Provider).WithField("name", i.plugin.Name()).Info("translated webhook succeeded!")

	return resp, nil
}

func (i *impl) CreateUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.CreateUser", attribute.String("psp", i.plugin.Name()))
	defer span.End()

	i.logger.WithField("name", i.plugin.Name()).Info("creating user...")

	resp, err := i.plugin.CreateUser(ctx, req)
	if err != nil {
		i.logger.WithField("name", i.plugin.Name()).Error("creating user failed:", err)
		otel.RecordError(span, err)
		return models.CreateUserResponse{}, translateError(err)
	}

	i.logger.WithField("name", i.plugin.Name()).Info("created user succeeded!")

	return resp, nil
}

func (i *impl) CreateUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.CreateUser", attribute.String("psp", i.plugin.Name()))
	defer span.End()

	i.logger.WithField("name", i.plugin.Name()).Info("creating user link...")

	resp, err := i.plugin.CreateUserLink(ctx, req)
	if err != nil {
		i.logger.WithField("name", i.plugin.Name()).Error("creating user link failed:", err)
		otel.RecordError(span, err)
		return models.CreateUserLinkResponse{}, translateError(err)
	}

	i.logger.WithField("name", i.plugin.Name()).Info("created user link succeeded!")

	return resp, nil
}

func (i *impl) CompleteUserLink(ctx context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.CompleteUserLink", attribute.String("psp", i.plugin.Name()))
	defer span.End()

	i.logger.WithField("name", i.plugin.Name()).Info("completing user link...")

	resp, err := i.plugin.CompleteUserLink(ctx, req)
	if err != nil {
		i.logger.WithField("name", i.plugin.Name()).Error("completing user link failed:", err)
		otel.RecordError(span, err)
		return models.CompleteUserLinkResponse{}, translateError(err)
	}

	i.logger.WithField("name", i.plugin.Name()).Info("completed user link succeeded!")

	return resp, nil
}

func (i *impl) UpdateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.UpdateUserLink", attribute.String("psp", i.plugin.Name()))
	defer span.End()

	i.logger.WithField("name", i.plugin.Name()).Info("updating user link...")

	resp, err := i.plugin.UpdateUserLink(ctx, req)
	if err != nil {
		i.logger.WithField("name", i.plugin.Name()).Error("updating user link failed:", err)
		otel.RecordError(span, err)
		return models.UpdateUserLinkResponse{}, translateError(err)
	}

	i.logger.WithField("name", i.plugin.Name()).Info("updated user link succeeded!")

	return resp, nil
}

func (i *impl) CompleteUpdateUserLink(ctx context.Context, req models.CompleteUpdateUserLinkRequest) (models.CompleteUpdateUserLinkResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.CompleteUpdateUserLink", attribute.String("psp", i.plugin.Name()))
	defer span.End()

	i.logger.WithField("name", i.plugin.Name()).Info("completing update user link...")

	resp, err := i.plugin.CompleteUpdateUserLink(ctx, req)
	if err != nil {
		i.logger.WithField("name", i.plugin.Name()).Error("completing update user link failed:", err)
		otel.RecordError(span, err)
		return models.CompleteUpdateUserLinkResponse{}, translateError(err)
	}

	i.logger.WithField("name", i.plugin.Name()).Info("completed update user link succeeded!")

	return resp, nil
}

func (i *impl) DeleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.DeleteUserConnection", attribute.String("psp", i.plugin.Name()))
	defer span.End()

	i.logger.WithField("name", i.plugin.Name()).Info("deleting user consent...")

	resp, err := i.plugin.DeleteUserConnection(ctx, req)
	if err != nil {
		i.logger.WithField("name", i.plugin.Name()).Error("deleting user consent failed:", err)
		otel.RecordError(span, err)
		return models.DeleteUserConnectionResponse{}, translateError(err)
	}

	i.logger.WithField("name", i.plugin.Name()).Info("deleted user consent succeeded!")

	return resp, nil
}

func (i *impl) DeleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	ctx, span := otel.StartSpan(ctx, "plugin.DeleteUser", attribute.String("psp", i.plugin.Name()))
	defer span.End()

	i.logger.WithField("name", i.plugin.Name()).Info("deleting user...")

	resp, err := i.plugin.DeleteUser(ctx, req)
	if err != nil {
		i.logger.WithField("name", i.plugin.Name()).Error("deleting user failed:", err)
		otel.RecordError(span, err)
		return models.DeleteUserResponse{}, translateError(err)
	}

	i.logger.WithField("name", i.plugin.Name()).Info("deleted user succeeded!")

	return resp, nil
}

var _ models.Plugin = &impl{}
