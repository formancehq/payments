package testserver

import (
	"context"
	"fmt"
	"net/http"
)

func pathPrefix(version int, path string) string {
	if version < 3 {
		return fmt.Sprintf("/%s", path)
	}
	return fmt.Sprintf("/v%d/%s", version, path)
}

func ConnectorInstall(ctx context.Context, srv *Server, ver int, reqBody any, res any) error {
	path := "connectors/install/generic"
	if ver == 2 {
		path = "connectors/generic"
	}
	return srv.Client().Do(ctx, http.MethodPost, pathPrefix(ver, path), reqBody, res)
}

func ConnectorUninstall(ctx context.Context, srv *Server, ver int, id string, res any) error {
	path := "connectors/" + id
	if ver == 2 {
		path = "connectors/generic/" + id
	}
	return srv.Client().Do(ctx, http.MethodDelete, pathPrefix(ver, path), nil, res)
}

func ConnectorConfig(ctx context.Context, srv *Server, ver int, id string, res any) error {
	path := "connectors/" + id + "/config"
	if ver == 2 {
		path = "connectors/generic/" + id + "/config"
	}
	return srv.Client().Get(ctx, pathPrefix(ver, path), res)
}

func ConnectorSchedules(ctx context.Context, srv *Server, ver int, id string, res any) error {
	path := "connectors/" + id + "/schedules"
	if ver == 2 {
		path = "connectors/generic/" + id + "/schedules"
	}
	return srv.Client().Get(ctx, pathPrefix(ver, path), res)
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
