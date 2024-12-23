package main

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"text/template"
)

var (
	//go:embed template/client/client.gotpl
	client string

	//go:embed template/client/accounts.gotpl
	clientAccounts string

	//go:embed template/client/balances.gotpl
	clientBalances string

	//go:embed template/client/external_accounts.gotpl
	clientExternalAccounts string

	//go:embed template/client/transactions.gotpl
	clientTransactions string

	//go:embed template/client/payouts.gotpl
	clientPayouts string

	//go:embed template/client/transfers.gotpl
	clientTransfers string
)

var (
	//go:embed template/accounts.gotpl
	accounts string

	//go:embed template/balances.gotpl
	balances string

	//go:embed template/capabilities.gotpl
	capabilities string

	//go:embed template/config.gotpl
	config string

	//go:embed template/currencies.gotpl
	currencies string

	//go:embed template/external_accounts.gotpl
	externalAccounts string

	//go:embed template/payments.gotpl
	payments string

	//go:embed template/payouts.gotpl
	payouts string

	//go:embed template/plugin.gotpl
	plugin string

	//go:embed template/transfers.gotpl
	transfers string

	//go:embed template/utils.gotpl
	utils string

	//go:embed template/workflow.gotpl
	workflow string
)

func createFiles(ctx context.Context, directoryPath string, params map[string]interface{}) error {
	files := map[string]string{
		"client/client.go":            client,
		"client/accounts.go":          clientAccounts,
		"client/balances.go":          clientBalances,
		"client/external_accounts.go": clientExternalAccounts,
		"client/transactions.go":      clientTransactions,
		"client/payouts.go":           clientPayouts,
		"client/tranfers.go":          clientTransfers,
		"accounts.go":                 accounts,
		"balances.go":                 balances,
		"capabilities.go":             capabilities,
		"config.go":                   config,
		"currencies.go":               currencies,
		"external_accounts.go":        externalAccounts,
		"payments.go":                 payments,
		"payouts.go":                  payouts,
		"plugin.go":                   plugin,
		"transfers.go":                transfers,
		"utils.go":                    utils,
		"workflow.go":                 workflow,
	}

	for path, tpl := range files {
		if err := createFile(
			ctx,
			filepath.Join(directoryPath, path),
			tpl,
			params,
		); err != nil {
			return err
		}
	}

	return nil
}

func createFile(ctx context.Context, path string, tpl string, params map[string]interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := template.New(path).Parse(tpl)
	if err != nil {
		return err
	}

	return t.Execute(f, params)
}
