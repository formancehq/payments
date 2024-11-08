package testserver

import (
	"context"

	v3 "github.com/formancehq/payments/internal/api/v3"
)

func CreateBankAccount(ctx context.Context, srv *Server, request v3.BankAccountsCreateRequest, res any) error {
	return srv.Client().Post(ctx, "/v3/bank-accounts", request, res)
}

func GetBankAccount(ctx context.Context, srv *Server, id string, res any) error {
	return srv.Client().Get(ctx, "/v3/bank-accounts/"+id, res)
}
