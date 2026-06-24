//go:build ee

package registry

import (
	bankingbridge "github.com/formancehq/payments/ee/plugins/bankingbridge"
	bitstamp "github.com/formancehq/payments/ee/plugins/bitstamp"
	coinbaseprime "github.com/formancehq/payments/ee/plugins/coinbaseprime"
	fireblocks "github.com/formancehq/payments/ee/plugins/fireblocks"
	routable "github.com/formancehq/payments/ee/plugins/routable"
	adyen "github.com/formancehq/payments/internal/connectors/plugins/public/adyen"
	atlar "github.com/formancehq/payments/internal/connectors/plugins/public/atlar"
	bankingcircle "github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle"
	column "github.com/formancehq/payments/internal/connectors/plugins/public/column"
	currencycloud "github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud"
	dummypay "github.com/formancehq/payments/internal/connectors/plugins/public/dummypay"
	generic "github.com/formancehq/payments/internal/connectors/plugins/public/generic"
	increase "github.com/formancehq/payments/internal/connectors/plugins/public/increase"
	mangopay "github.com/formancehq/payments/internal/connectors/plugins/public/mangopay"
	modulr "github.com/formancehq/payments/internal/connectors/plugins/public/modulr"
	moneycorp "github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp"
	plaid "github.com/formancehq/payments/internal/connectors/plugins/public/plaid"
	powens "github.com/formancehq/payments/internal/connectors/plugins/public/powens"
	qonto "github.com/formancehq/payments/internal/connectors/plugins/public/qonto"
	stripe "github.com/formancehq/payments/internal/connectors/plugins/public/stripe"
	tink "github.com/formancehq/payments/internal/connectors/plugins/public/tink"
	wise "github.com/formancehq/payments/internal/connectors/plugins/public/wise"
	pkgplugins "github.com/formancehq/payments/pkg/domain/plugins"
)

func init() {
	load(map[string]pkgplugins.Registration{
		adyen.ProviderName:         adyen.Registration,
		atlar.ProviderName:         atlar.Registration,
		bankingcircle.ProviderName: bankingcircle.Registration,
		column.ProviderName:        column.Registration,
		currencycloud.ProviderName: currencycloud.Registration,
		DummyPSPName:               dummypay.Registration,
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
		bankingbridge.ProviderName: bankingbridge.Registration,
		bitstamp.ProviderName:      bitstamp.Registration,
		coinbaseprime.ProviderName: coinbaseprime.Registration,
		fireblocks.ProviderName:    fireblocks.Registration,
		routable.ProviderName:      routable.Registration,
	})
}
