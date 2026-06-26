//go:build !ee

package registry

import (
	adyen "github.com/formancehq/payments/ce/plugins/adyen"
	atlar "github.com/formancehq/payments/ce/plugins/atlar"
	bankingcircle "github.com/formancehq/payments/ce/plugins/bankingcircle"
	column "github.com/formancehq/payments/ce/plugins/column"
	currencycloud "github.com/formancehq/payments/ce/plugins/currencycloud"
	generic "github.com/formancehq/payments/ce/plugins/generic"
	increase "github.com/formancehq/payments/ce/plugins/increase"
	mangopay "github.com/formancehq/payments/ce/plugins/mangopay"
	modulr "github.com/formancehq/payments/ce/plugins/modulr"
	moneycorp "github.com/formancehq/payments/ce/plugins/moneycorp"
	plaid "github.com/formancehq/payments/ce/plugins/plaid"
	powens "github.com/formancehq/payments/ce/plugins/powens"
	qonto "github.com/formancehq/payments/ce/plugins/qonto"
	stripe "github.com/formancehq/payments/ce/plugins/stripe"
	tink "github.com/formancehq/payments/ce/plugins/tink"
	wise "github.com/formancehq/payments/ce/plugins/wise"
	dummypay "github.com/formancehq/payments/internal/connectors/plugins/public/dummypay"
	pkgplugins "github.com/formancehq/payments/pkg/domain/plugins"
)

func init() {
	load(map[string]pkgplugins.Registration{
		adyen.ProviderName:         adyen.Registration,
		atlar.ProviderName:         atlar.Registration,
		bankingcircle.ProviderName: bankingcircle.Registration,
		column.ProviderName:        column.Registration,
		currencycloud.ProviderName: currencycloud.Registration,
		generic.ProviderName:       generic.Registration,
		increase.ProviderName:      increase.Registration,
		mangopay.ProviderName:      mangopay.Registration,
		modulr.ProviderName:        modulr.Registration,
		moneycorp.ProviderName:     moneycorp.Registration,
		plaid.ProviderName:         plaid.Registration,
		powens.ProviderName:        powens.Registration,
		qonto.ProviderName:         qonto.Registration,
		stripe.ProviderName:        stripe.Registration,
		tink.ProviderName:          tink.Registration,
		wise.ProviderName:          wise.Registration,
		DummyPSPName:               dummypay.Registration,
	})
}
