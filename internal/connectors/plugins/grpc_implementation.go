package plugins

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/grpc"
	"github.com/formancehq/payments/internal/connectors/grpc/proto"
	"github.com/formancehq/payments/internal/connectors/grpc/proto/services"
	"github.com/formancehq/payments/internal/models"
	"github.com/hashicorp/go-hclog"
)

type impl struct {
	plugin models.Plugin
}

func NewGRPCImplem(plugin models.Plugin) *impl {
	return &impl{
		plugin: plugin,
	}
}

func (i *impl) Install(ctx context.Context, req *services.InstallRequest) (*services.InstallResponse, error) {
	hclog.Default().Info("installing...")

	resp, err := i.plugin.Install(ctx, models.InstallRequest{
		Config: req.Config,
	})
	if err != nil {
		hclog.Default().Error("install failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	capabilities := make([]proto.Capability, 0, len(resp.Capabilities))
	for _, capability := range resp.Capabilities {
		capabilities = append(capabilities, proto.Capability(capability))
	}

	webhooksConfigs := make([]*proto.WebhookConfig, 0, len(resp.WebhooksConfigs))
	for _, webhook := range resp.WebhooksConfigs {
		webhooksConfigs = append(webhooksConfigs, &proto.WebhookConfig{
			Name:    webhook.Name,
			UrlPath: webhook.URLPath,
		})
	}

	hclog.Default().Info("installed!")

	return &services.InstallResponse{
		Capabilities:    capabilities,
		Workflow:        grpc.TranslateWorkflow(resp.Workflow),
		WebhooksConfigs: webhooksConfigs,
	}, nil
}

func (i *impl) Uninstall(ctx context.Context, req *services.UninstallRequest) (*services.UninstallResponse, error) {
	hclog.Default().Info("uninstalling...")

	_, err := i.plugin.Uninstall(ctx, models.UninstallRequest{
		ConnectorID: req.ConnectorId,
	})
	if err != nil {
		hclog.Default().Error("uninstall failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	hclog.Default().Info("uninstalled!")

	return &services.UninstallResponse{}, nil
}

func (i *impl) FetchNextAccounts(ctx context.Context, req *services.FetchNextAccountsRequest) (*services.FetchNextAccountsResponse, error) {
	hclog.Default().Info("fetching next accounts...")

	resp, err := i.plugin.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{
		FromPayload: req.FromPayload,
		State:       req.State,
		PageSize:    int(req.PageSize),
	})
	if err != nil {
		hclog.Default().Error("fetching next accounts failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	accounts := make([]*proto.Account, 0, len(resp.Accounts))
	for _, account := range resp.Accounts {
		accounts = append(accounts, grpc.TranslateAccount(account))
	}

	hclog.Default().Info("fetched next accounts succeeded!")

	return &services.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: resp.NewState,
		HasMore:  resp.HasMore,
	}, nil
}

func (i *impl) FetchNextExternalAccounts(ctx context.Context, req *services.FetchNextExternalAccountsRequest) (*services.FetchNextExternalAccountsResponse, error) {
	hclog.Default().Info("fetching next external accounts...")

	resp, err := i.plugin.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{
		FromPayload: req.FromPayload,
		State:       req.State,
		PageSize:    int(req.PageSize),
	})
	if err != nil {
		hclog.Default().Error("fetching next external accounts failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	externalAccounts := make([]*proto.Account, 0, len(resp.ExternalAccounts))
	for _, account := range resp.ExternalAccounts {
		externalAccounts = append(externalAccounts, grpc.TranslateAccount(account))
	}

	hclog.Default().Info("fetched next external accounts succeeded!")

	return &services.FetchNextExternalAccountsResponse{
		Accounts: externalAccounts,
		NewState: resp.NewState,
		HasMore:  resp.HasMore,
	}, nil
}

func (i *impl) FetchNextPayments(ctx context.Context, req *services.FetchNextPaymentsRequest) (*services.FetchNextPaymentsResponse, error) {
	hclog.Default().Info("fetching next payments...")

	resp, err := i.plugin.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{
		FromPayload: req.FromPayload,
		State:       req.State,
		PageSize:    int(req.PageSize),
	})
	if err != nil {
		hclog.Default().Error("fetching next payments failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	payments := make([]*proto.Payment, 0, len(resp.Payments))
	for _, payment := range resp.Payments {
		payments = append(payments, grpc.TranslatePayment(payment))
	}

	hclog.Default().Info("fetched next payments succeeded!")

	return &services.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: resp.NewState,
		HasMore:  resp.HasMore,
	}, nil
}

func (i *impl) FetchNextBalances(ctx context.Context, req *services.FetchNextBalancesRequest) (*services.FetchNextBalancesResponse, error) {
	hclog.Default().Info("fetching next balances...")

	resp, err := i.plugin.FetchNextBalances(ctx, models.FetchNextBalancesRequest{
		FromPayload: req.FromPayload,
		State:       req.State,
		PageSize:    int(req.PageSize),
	})
	if err != nil {
		hclog.Default().Error("fetching next balances failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	balances := make([]*proto.Balance, 0, len(resp.Balances))
	for _, balance := range resp.Balances {
		balances = append(balances, grpc.TranslateBalance(balance))
	}

	hclog.Default().Info("fetched next balances succeeded!")

	return &services.FetchNextBalancesResponse{
		Balances: balances,
		NewState: resp.NewState,
		HasMore:  resp.HasMore,
	}, nil
}

func (i *impl) FetchNextOthers(ctx context.Context, req *services.FetchNextOthersRequest) (*services.FetchNextOthersResponse, error) {
	hclog.Default().Info("fetching next others...")

	resp, err := i.plugin.FetchNextOthers(ctx, models.FetchNextOthersRequest{
		FromPayload: req.FromPayload,
		State:       req.State,
		PageSize:    int(req.PageSize),
		Name:        req.Name,
	})
	if err != nil {
		hclog.Default().Error("fetching next others failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	others := make([]*proto.Other, 0, len(resp.Others))
	for _, other := range resp.Others {
		others = append(others, &proto.Other{
			Id:    other.ID,
			Other: other.Other,
		})
	}

	hclog.Default().Info("fetched next others succeeded!")

	return &services.FetchNextOthersResponse{
		Others:   others,
		NewState: resp.NewState,
		HasMore:  resp.HasMore,
	}, nil
}

func (i *impl) CreateBankAccount(ctx context.Context, req *services.CreateBankAccountRequest) (*services.CreateBankAccountResponse, error) {
	hclog.Default().Info("creating bank account...")

	resp, err := i.plugin.CreateBankAccount(ctx, models.CreateBankAccountRequest{
		BankAccount: grpc.TranslateProtoBankAccount(req.BankAccount),
	})
	if err != nil {
		hclog.Default().Error("creating bank account failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	hclog.Default().Info("created bank account succeeded!")

	return &services.CreateBankAccountResponse{
		RelatedAccount: grpc.TranslateAccount(resp.RelatedAccount),
	}, nil
}

func (i *impl) CreateTransfer(ctx context.Context, req *services.CreateTransferRequest) (*services.CreateTransferResponse, error) {
	hclog.Default().Info("creating transfer...")

	pi, err := grpc.TranslateProtoPaymentInitiation(req.PaymentInitiation)
	if err != nil {
		hclog.Default().Error("creating transfer failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	resp, err := i.plugin.CreateTransfer(ctx, models.CreateTransferRequest{
		PaymentInitiation: pi,
	})
	if err != nil {
		hclog.Default().Error("creating transfer failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	hclog.Default().Info("created transfer succeeded!")

	return &services.CreateTransferResponse{
		Payment: grpc.TranslatePayment(resp.Payment),
	}, nil
}

func (i *impl) CreatePayout(ctx context.Context, req *services.CreatePayoutRequest) (*services.CreatePayoutResponse, error) {
	hclog.Default().Info("creating payout...")

	pi, err := grpc.TranslateProtoPaymentInitiation(req.PaymentInitiation)
	if err != nil {
		hclog.Default().Error("creating payout failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	resp, err := i.plugin.CreatePayout(ctx, models.CreatePayoutRequest{
		PaymentInitiation: pi,
	})
	if err != nil {
		hclog.Default().Error("creating payout failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	hclog.Default().Info("created payout succeeded!")

	return &services.CreatePayoutResponse{
		Payment: grpc.TranslatePayment(resp.Payment),
	}, nil
}

func (i *impl) CreateWebhooks(ctx context.Context, req *services.CreateWebhooksRequest) (*services.CreateWebhooksResponse, error) {
	hclog.Default().Info("creating webhooks...")

	resp, err := i.plugin.CreateWebhooks(ctx, models.CreateWebhooksRequest{
		ConnectorID:    req.ConnectorId,
		FromPayload:    req.FromPayload,
		WebhookBaseUrl: req.WebhookBaseUrl,
	})
	if err != nil {
		hclog.Default().Error("creating webhooks failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	hclog.Default().Info("created webhooks succeeded!")

	others := make([]*proto.Other, 0, len(resp.Others))
	for _, other := range resp.Others {
		others = append(others, &proto.Other{
			Id:    other.ID,
			Other: other.Other,
		})
	}

	return &services.CreateWebhooksResponse{
		Others: others,
	}, nil
}

func (i *impl) TranslateWebhook(ctx context.Context, req *services.TranslateWebhookRequest) (*services.TranslateWebhookResponse, error) {
	hclog.Default().Info("translating webhook...")

	resp, err := i.plugin.TranslateWebhook(ctx, models.TranslateWebhookRequest{
		Name:    req.Name,
		Webhook: grpc.TranslateProtoWebhook(req.Webhook),
	})
	if err != nil {
		hclog.Default().Error("translating webhook failed: ", err)
		return nil, translateErrorToGRPC(err)
	}

	hclog.Default().Info("translated webhook succeeded!")

	responses := make([]*services.TranslateWebhookResponse_Response, 0, len(resp.Responses))
	for _, response := range resp.Responses {
		r := &services.TranslateWebhookResponse_Response{
			IdempotencyKey: response.IdempotencyKey,
		}

		if response.Account != nil {
			r.Translated = &services.TranslateWebhookResponse_Response_Account{
				Account: grpc.TranslateAccount(*response.Account),
			}
		}

		if response.ExternalAccount != nil {
			r.Translated = &services.TranslateWebhookResponse_Response_ExternalAccount{
				ExternalAccount: grpc.TranslateAccount(*response.ExternalAccount),
			}
		}

		if response.Payment != nil {
			r.Translated = &services.TranslateWebhookResponse_Response_Payment{
				Payment: grpc.TranslatePayment(*response.Payment),
			}
		}

		responses = append(responses, r)
	}

	return &services.TranslateWebhookResponse{
		Responses: responses,
	}, nil
}

var _ grpc.PSP = &impl{}
