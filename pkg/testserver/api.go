package testserver

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/formancehq/payments/internal/api/v2"
)

func pathPrefix(version int, path string) string {
	if version < 3 {
		return fmt.Sprintf("/%s", path)
	}
	return fmt.Sprintf("/v%d/%s", version, path)
}

func ConnectorInstall(ctx context.Context, srv *Server, ver int, reqBody any, res any) error {
	path := "connectors/install/dummypay"
	if ver == 2 {
		path = "connectors/dummypay"
	}
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, path), reqBody, res)
}

func ConnectorConfigUpdate(ctx context.Context, srv *Server, ver int, id string, reqBody any) error {
	path := "connectors/" + id + "/config"
	method := http.MethodPatch
	if ver == 2 {
		path = "connectors/dummypay/" + id + "/config"
		method = http.MethodPost
	}
	return srv.Client().Do(ctx, method, pathPrefix(ver, path), reqBody, nil)
}

func ConnectorConfigs(ctx context.Context, srv *Server, ver int, res any) error {
	path := "connectors/configs"
	return srv.Client().Get(ctx, pathPrefix(ver, path), res)
}

func ConnectorConfig(ctx context.Context, srv *Server, ver int, id string, res any) error {
	path := "connectors/" + id + "/config"
	if ver == 2 {
		path = "connectors/dummypay/" + id + "/config"
	}
	return srv.Client().Get(ctx, pathPrefix(ver, path), res)
}

func CreateAccount(ctx context.Context, srv *Server, ver int, reqBody any, res any) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "accounts"), reqBody, res)
}

func ListAccounts(ctx context.Context, srv *Server, ver int, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "accounts"), res)
}

func GetAccount(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "accounts/"+id), res)
}

func GetAccountBalances(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "accounts/"+id+"/balances"), res)
}

func CreateBankAccount(ctx context.Context, srv *Server, ver int, reqBody any, res any) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "bank-accounts"), reqBody, res)
}

func GetBankAccount(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "bank-accounts/"+id), res)
}

func ForwardBankAccount(ctx context.Context, srv *Server, ver int, id string, reqBody any, res any) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "bank-accounts/"+id+"/forward"), reqBody, res)
}

func UpdateBankAccountMetadata(ctx context.Context, srv *Server, ver int, id string, reqBody any, res any) error {
	return srv.Client().Do(ctx, http.MethodPatch, pathPrefix(ver, "bank-accounts/"+id+"/metadata"), reqBody, res)
}

func CreatePayment(ctx context.Context, srv *Server, ver int, reqBody any, res any) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "payments"), reqBody, res)
}

func GetPayment(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "payments/"+id), res)
}

func CreatePaymentInitiation(ctx context.Context, srv *Server, ver int, reqBody any, res any) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "payment-initiations"), reqBody, res)
}

func GetPaymentInitiation(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "payment-initiations/"+id), res)
}

func ApprovePaymentInitiation(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "payment-initiations/"+id+"/approve"), nil, res)
}

func RejectPaymentInitiation(ctx context.Context, srv *Server, ver int, id string) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "payment-initiations/"+id+"/reject"), nil, nil)
}

func ReversePaymentInitiation(ctx context.Context, srv *Server, ver int, id string, reqBody any, res any) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "payment-initiations/"+id+"/reverse"), reqBody, res)
}

func CreatePool(ctx context.Context, srv *Server, ver int, reqBody any, res any) error {
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "pools"), reqBody, res)
}

func RemovePool(ctx context.Context, srv *Server, ver int, id string) error {
	return srv.Client().Do(ctx, http.MethodDelete, pathPrefix(ver, "pools/"+id), nil, nil)
}

func GetPool(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "pools/"+id), res)
}

func RemovePoolAccount(ctx context.Context, srv *Server, ver int, id string, accountID string) error {
	return srv.Client().Do(ctx, http.MethodDelete, pathPrefix(ver, "pools/"+id+"/accounts/"+accountID), nil, nil)
}

func AddPoolAccount(ctx context.Context, srv *Server, ver int, id string, accountID string) error {
	if ver == 2 {
		req := v2.PoolsAddAccountRequest{AccountID: accountID}
		return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "pools/"+id+"/accounts"), &req, nil)
	}
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, "pools/"+id+"/accounts/"+accountID), nil, nil)
}

func GetTask(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "tasks/"+id), res)
}
