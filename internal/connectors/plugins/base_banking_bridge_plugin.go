package plugins

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

type BaseBankingBridgePlugin struct{}

func NewBaseBankingBridgePlugin() *BaseBankingBridgePlugin {
	return &BaseBankingBridgePlugin{}
}

func (dp *BaseBankingBridgePlugin) Name() string {
	return "default"
}

func (dp *BaseBankingBridgePlugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{}, ErrNotImplemented
}

func (dp *BaseBankingBridgePlugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, ErrNotImplemented
}

func (dp *BaseBankingBridgePlugin) CreateUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	return models.CreateUserLinkResponse{}, ErrNotImplemented
}

var _ models.BankingBridgePlugin = &BaseBankingBridgePlugin{}
