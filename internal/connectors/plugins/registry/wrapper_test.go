package registry

import (
	"context"
	"testing"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInstall(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().Install(gomock.Any(), models.InstallRequest{}).Return(models.InstallResponse{}, nil)
		resp, err := i.Install(ctx, models.InstallRequest{})
		require.Nil(t, err)
		require.Equal(t, models.InstallResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().Install(gomock.Any(), models.InstallRequest{}).Return(models.InstallResponse{}, plugins.ErrNotImplemented)
		resp, err := i.Install(ctx, models.InstallRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.InstallResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().Install(gomock.Any(), models.InstallRequest{}).Return(models.InstallResponse{}, models.ErrInvalidConfig)
		resp, err := i.Install(ctx, models.InstallRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.InstallResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().Install(gomock.Any(), models.InstallRequest{}).Return(models.InstallResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.Install(ctx, models.InstallRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.InstallResponse{}, resp)
	})
}

func TestUninstall(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().Uninstall(gomock.Any(), models.UninstallRequest{}).Return(models.UninstallResponse{}, nil)
		resp, err := i.Uninstall(ctx, models.UninstallRequest{})
		require.Nil(t, err)
		require.Equal(t, models.UninstallResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().Uninstall(gomock.Any(), models.UninstallRequest{}).Return(models.UninstallResponse{}, plugins.ErrNotImplemented)
		resp, err := i.Uninstall(ctx, models.UninstallRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.UninstallResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().Uninstall(gomock.Any(), models.UninstallRequest{}).Return(models.UninstallResponse{}, models.ErrInvalidConfig)
		resp, err := i.Uninstall(ctx, models.UninstallRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.UninstallResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().Uninstall(gomock.Any(), models.UninstallRequest{}).Return(models.UninstallResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.Uninstall(ctx, models.UninstallRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.UninstallResponse{}, resp)
	})
}

func TestFetchNextAccounts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextAccounts(gomock.Any(), models.FetchNextAccountsRequest{}).Return(models.FetchNextAccountsResponse{}, nil)
		resp, err := i.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
		require.Nil(t, err)
		require.Equal(t, models.FetchNextAccountsResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextAccounts(gomock.Any(), models.FetchNextAccountsRequest{}).Return(models.FetchNextAccountsResponse{}, plugins.ErrNotImplemented)
		resp, err := i.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.FetchNextAccountsResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextAccounts(gomock.Any(), models.FetchNextAccountsRequest{}).Return(models.FetchNextAccountsResponse{}, models.ErrInvalidConfig)
		resp, err := i.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.FetchNextAccountsResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextAccounts(gomock.Any(), models.FetchNextAccountsRequest{}).Return(models.FetchNextAccountsResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.FetchNextAccountsResponse{}, resp)
	})
}

func TestFetchNextExternalAccounts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextExternalAccounts(gomock.Any(), models.FetchNextExternalAccountsRequest{}).Return(models.FetchNextExternalAccountsResponse{}, nil)
		resp, err := i.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{})
		require.Nil(t, err)
		require.Equal(t, models.FetchNextExternalAccountsResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextExternalAccounts(gomock.Any(), models.FetchNextExternalAccountsRequest{}).Return(models.FetchNextExternalAccountsResponse{}, plugins.ErrNotImplemented)
		resp, err := i.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.FetchNextExternalAccountsResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextExternalAccounts(gomock.Any(), models.FetchNextExternalAccountsRequest{}).Return(models.FetchNextExternalAccountsResponse{}, models.ErrInvalidConfig)
		resp, err := i.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.FetchNextExternalAccountsResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextExternalAccounts(gomock.Any(), models.FetchNextExternalAccountsRequest{}).Return(models.FetchNextExternalAccountsResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.FetchNextExternalAccountsResponse{}, resp)
	})
}

func TestFetchNextPayments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextPayments(gomock.Any(), models.FetchNextPaymentsRequest{}).Return(models.FetchNextPaymentsResponse{}, nil)
		resp, err := i.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{})
		require.Nil(t, err)
		require.Equal(t, models.FetchNextPaymentsResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextPayments(gomock.Any(), models.FetchNextPaymentsRequest{}).Return(models.FetchNextPaymentsResponse{}, plugins.ErrNotImplemented)
		resp, err := i.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.FetchNextPaymentsResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextPayments(gomock.Any(), models.FetchNextPaymentsRequest{}).Return(models.FetchNextPaymentsResponse{}, models.ErrInvalidConfig)
		resp, err := i.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.FetchNextPaymentsResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextPayments(gomock.Any(), models.FetchNextPaymentsRequest{}).Return(models.FetchNextPaymentsResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.FetchNextPaymentsResponse{}, resp)
	})
}

func TestFetchNextBalances(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextBalances(gomock.Any(), models.FetchNextBalancesRequest{}).Return(models.FetchNextBalancesResponse{}, nil)
		resp, err := i.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		require.Nil(t, err)
		require.Equal(t, models.FetchNextBalancesResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextBalances(gomock.Any(), models.FetchNextBalancesRequest{}).Return(models.FetchNextBalancesResponse{}, plugins.ErrNotImplemented)
		resp, err := i.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.FetchNextBalancesResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextBalances(gomock.Any(), models.FetchNextBalancesRequest{}).Return(models.FetchNextBalancesResponse{}, models.ErrInvalidConfig)
		resp, err := i.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.FetchNextBalancesResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextBalances(gomock.Any(), models.FetchNextBalancesRequest{}).Return(models.FetchNextBalancesResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.FetchNextBalancesResponse{}, resp)
	})
}

func TestFetchNextOthers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextOthers(gomock.Any(), models.FetchNextOthersRequest{}).Return(models.FetchNextOthersResponse{}, nil)
		resp, err := i.FetchNextOthers(ctx, models.FetchNextOthersRequest{})
		require.Nil(t, err)
		require.Equal(t, models.FetchNextOthersResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextOthers(gomock.Any(), models.FetchNextOthersRequest{}).Return(models.FetchNextOthersResponse{}, plugins.ErrNotImplemented)
		resp, err := i.FetchNextOthers(ctx, models.FetchNextOthersRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.FetchNextOthersResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextOthers(gomock.Any(), models.FetchNextOthersRequest{}).Return(models.FetchNextOthersResponse{}, models.ErrInvalidConfig)
		resp, err := i.FetchNextOthers(ctx, models.FetchNextOthersRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.FetchNextOthersResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().FetchNextOthers(gomock.Any(), models.FetchNextOthersRequest{}).Return(models.FetchNextOthersResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.FetchNextOthers(ctx, models.FetchNextOthersRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.FetchNextOthersResponse{}, resp)
	})
}

func TestCreateBankAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateBankAccount(gomock.Any(), models.CreateBankAccountRequest{}).Return(models.CreateBankAccountResponse{}, nil)
		resp, err := i.CreateBankAccount(ctx, models.CreateBankAccountRequest{})
		require.Nil(t, err)
		require.Equal(t, models.CreateBankAccountResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateBankAccount(gomock.Any(), models.CreateBankAccountRequest{}).Return(models.CreateBankAccountResponse{}, plugins.ErrNotImplemented)
		resp, err := i.CreateBankAccount(ctx, models.CreateBankAccountRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.CreateBankAccountResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateBankAccount(gomock.Any(), models.CreateBankAccountRequest{}).Return(models.CreateBankAccountResponse{}, models.ErrInvalidConfig)
		resp, err := i.CreateBankAccount(ctx, models.CreateBankAccountRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.CreateBankAccountResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateBankAccount(gomock.Any(), models.CreateBankAccountRequest{}).Return(models.CreateBankAccountResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.CreateBankAccount(ctx, models.CreateBankAccountRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.CreateBankAccountResponse{}, resp)
	})
}

func TestCreateTransfer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateTransfer(gomock.Any(), models.CreateTransferRequest{}).Return(models.CreateTransferResponse{}, nil)
		resp, err := i.CreateTransfer(ctx, models.CreateTransferRequest{})
		require.Nil(t, err)
		require.Equal(t, models.CreateTransferResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateTransfer(gomock.Any(), models.CreateTransferRequest{}).Return(models.CreateTransferResponse{}, plugins.ErrNotImplemented)
		resp, err := i.CreateTransfer(ctx, models.CreateTransferRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.CreateTransferResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateTransfer(gomock.Any(), models.CreateTransferRequest{}).Return(models.CreateTransferResponse{}, models.ErrInvalidConfig)
		resp, err := i.CreateTransfer(ctx, models.CreateTransferRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.CreateTransferResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateTransfer(gomock.Any(), models.CreateTransferRequest{}).Return(models.CreateTransferResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.CreateTransfer(ctx, models.CreateTransferRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.CreateTransferResponse{}, resp)
	})
}

func TestReverseTransfer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().ReverseTransfer(gomock.Any(), models.ReverseTransferRequest{}).Return(models.ReverseTransferResponse{}, nil)
		resp, err := i.ReverseTransfer(ctx, models.ReverseTransferRequest{})
		require.Nil(t, err)
		require.Equal(t, models.ReverseTransferResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().ReverseTransfer(gomock.Any(), models.ReverseTransferRequest{}).Return(models.ReverseTransferResponse{}, plugins.ErrNotImplemented)
		resp, err := i.ReverseTransfer(ctx, models.ReverseTransferRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.ReverseTransferResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().ReverseTransfer(gomock.Any(), models.ReverseTransferRequest{}).Return(models.ReverseTransferResponse{}, models.ErrInvalidConfig)
		resp, err := i.ReverseTransfer(ctx, models.ReverseTransferRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.ReverseTransferResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().ReverseTransfer(gomock.Any(), models.ReverseTransferRequest{}).Return(models.ReverseTransferResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.ReverseTransfer(ctx, models.ReverseTransferRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.ReverseTransferResponse{}, resp)
	})
}

func TestPollTransferStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().PollTransferStatus(gomock.Any(), models.PollTransferStatusRequest{}).Return(models.PollTransferStatusResponse{}, nil)
		resp, err := i.PollTransferStatus(ctx, models.PollTransferStatusRequest{})
		require.Nil(t, err)
		require.Equal(t, models.PollTransferStatusResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().PollTransferStatus(gomock.Any(), models.PollTransferStatusRequest{}).Return(models.PollTransferStatusResponse{}, plugins.ErrNotImplemented)
		resp, err := i.PollTransferStatus(ctx, models.PollTransferStatusRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.PollTransferStatusResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().PollTransferStatus(gomock.Any(), models.PollTransferStatusRequest{}).Return(models.PollTransferStatusResponse{}, models.ErrInvalidConfig)
		resp, err := i.PollTransferStatus(ctx, models.PollTransferStatusRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.PollTransferStatusResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().PollTransferStatus(gomock.Any(), models.PollTransferStatusRequest{}).Return(models.PollTransferStatusResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.PollTransferStatus(ctx, models.PollTransferStatusRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.PollTransferStatusResponse{}, resp)
	})
}

func TestCreatePayout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreatePayout(gomock.Any(), models.CreatePayoutRequest{}).Return(models.CreatePayoutResponse{}, nil)
		resp, err := i.CreatePayout(ctx, models.CreatePayoutRequest{})
		require.Nil(t, err)
		require.Equal(t, models.CreatePayoutResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreatePayout(gomock.Any(), models.CreatePayoutRequest{}).Return(models.CreatePayoutResponse{}, plugins.ErrNotImplemented)
		resp, err := i.CreatePayout(ctx, models.CreatePayoutRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.CreatePayoutResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreatePayout(gomock.Any(), models.CreatePayoutRequest{}).Return(models.CreatePayoutResponse{}, models.ErrInvalidConfig)
		resp, err := i.CreatePayout(ctx, models.CreatePayoutRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.CreatePayoutResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreatePayout(gomock.Any(), models.CreatePayoutRequest{}).Return(models.CreatePayoutResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.CreatePayout(ctx, models.CreatePayoutRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.CreatePayoutResponse{}, resp)
	})
}

func TestReversePayout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().ReversePayout(gomock.Any(), models.ReversePayoutRequest{}).Return(models.ReversePayoutResponse{}, nil)
		resp, err := i.ReversePayout(ctx, models.ReversePayoutRequest{})
		require.Nil(t, err)
		require.Equal(t, models.ReversePayoutResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().ReversePayout(gomock.Any(), models.ReversePayoutRequest{}).Return(models.ReversePayoutResponse{}, plugins.ErrNotImplemented)
		resp, err := i.ReversePayout(ctx, models.ReversePayoutRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.ReversePayoutResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().ReversePayout(gomock.Any(), models.ReversePayoutRequest{}).Return(models.ReversePayoutResponse{}, models.ErrInvalidConfig)
		resp, err := i.ReversePayout(ctx, models.ReversePayoutRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.ReversePayoutResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().ReversePayout(gomock.Any(), models.ReversePayoutRequest{}).Return(models.ReversePayoutResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.ReversePayout(ctx, models.ReversePayoutRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.ReversePayoutResponse{}, resp)
	})
}

func TestPollPayoutStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().PollPayoutStatus(gomock.Any(), models.PollPayoutStatusRequest{}).Return(models.PollPayoutStatusResponse{}, nil)
		resp, err := i.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{})
		require.Nil(t, err)
		require.Equal(t, models.PollPayoutStatusResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().PollPayoutStatus(gomock.Any(), models.PollPayoutStatusRequest{}).Return(models.PollPayoutStatusResponse{}, plugins.ErrNotImplemented)
		resp, err := i.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.PollPayoutStatusResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().PollPayoutStatus(gomock.Any(), models.PollPayoutStatusRequest{}).Return(models.PollPayoutStatusResponse{}, models.ErrInvalidConfig)
		resp, err := i.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.PollPayoutStatusResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().PollPayoutStatus(gomock.Any(), models.PollPayoutStatusRequest{}).Return(models.PollPayoutStatusResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.PollPayoutStatusResponse{}, resp)
	})
}

func TestCreateWebhooks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateWebhooks(gomock.Any(), models.CreateWebhooksRequest{}).Return(models.CreateWebhooksResponse{}, nil)
		resp, err := i.CreateWebhooks(ctx, models.CreateWebhooksRequest{})
		require.Nil(t, err)
		require.Equal(t, models.CreateWebhooksResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateWebhooks(gomock.Any(), models.CreateWebhooksRequest{}).Return(models.CreateWebhooksResponse{}, plugins.ErrNotImplemented)
		resp, err := i.CreateWebhooks(ctx, models.CreateWebhooksRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.CreateWebhooksResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateWebhooks(gomock.Any(), models.CreateWebhooksRequest{}).Return(models.CreateWebhooksResponse{}, models.ErrInvalidConfig)
		resp, err := i.CreateWebhooks(ctx, models.CreateWebhooksRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.CreateWebhooksResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().CreateWebhooks(gomock.Any(), models.CreateWebhooksRequest{}).Return(models.CreateWebhooksResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.CreateWebhooks(ctx, models.CreateWebhooksRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.CreateWebhooksResponse{}, resp)
	})
}

func TestTranslateWebhooks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := models.NewMockPlugin(ctrl)
	i := New(logging.Testing(), m)

	t.Run("success", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().TranslateWebhook(gomock.Any(), models.TranslateWebhookRequest{}).Return(models.TranslateWebhookResponse{}, nil)
		resp, err := i.TranslateWebhook(ctx, models.TranslateWebhookRequest{})
		require.Nil(t, err)
		require.Equal(t, models.TranslateWebhookResponse{}, resp)
	})

	t.Run("error not implemented", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().TranslateWebhook(gomock.Any(), models.TranslateWebhookRequest{}).Return(models.TranslateWebhookResponse{}, plugins.ErrNotImplemented)
		resp, err := i.TranslateWebhook(ctx, models.TranslateWebhookRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrNotImplemented)
		require.Equal(t, models.TranslateWebhookResponse{}, resp)
	})

	t.Run("error invalid config", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().TranslateWebhook(gomock.Any(), models.TranslateWebhookRequest{}).Return(models.TranslateWebhookResponse{}, models.ErrInvalidConfig)
		resp, err := i.TranslateWebhook(ctx, models.TranslateWebhookRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrInvalidClientRequest)
		require.ErrorIs(t, err, models.ErrInvalidConfig)
		require.Equal(t, models.TranslateWebhookResponse{}, resp)
	})

	t.Run("error too many requests", func(t *testing.T) {
		m.EXPECT().Name().Return("test")
		m.EXPECT().TranslateWebhook(gomock.Any(), models.TranslateWebhookRequest{}).Return(models.TranslateWebhookResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
		resp, err := i.TranslateWebhook(ctx, models.TranslateWebhookRequest{})
		require.NotNil(t, err)
		require.ErrorIs(t, err, plugins.ErrUpstreamRatelimit)
		require.ErrorIs(t, err, httpwrapper.ErrStatusCodeTooManyRequests)
		require.Equal(t, models.TranslateWebhookResponse{}, resp)
	})
}
