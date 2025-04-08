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
