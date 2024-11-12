package testserver

import (
	"context"
	"fmt"
)

func pathPrefix(version int, path string) string {
	if version < 3 {
		return fmt.Sprintf("/%s", path)
	}
	return fmt.Sprintf("/v%d/%s", version, path)
}

func CreateBankAccount(ctx context.Context, srv *Server, ver int, reqBody any, res any) error {
	return srv.Client().Post(ctx, pathPrefix(ver, "bank-accounts"), reqBody, res)
}

func GetBankAccount(ctx context.Context, srv *Server, ver int, id string, res any) error {
	return srv.Client().Get(ctx, pathPrefix(ver, "bank-accounts/"+id), res)
}

func ForwardBankAccount(ctx context.Context, srv *Server, ver int, id string, reqBody any, res any) error {
	return srv.Client().Post(ctx, pathPrefix(ver, "bank-accounts/"+id+"/forward"), reqBody, res)
}
