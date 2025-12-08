<p align="center">
  <img src="https://formance01.b-cdn.net/Github-Attachements/banners/connectivity-readme-banner.webp" alt="connectivity" width="100%" />
</p>

# Formance Payments [![test](https://github.com/formancehq/payments/actions/workflows/main.yml/badge.svg)](https://github.com/formancehq/payments/actions/workflows/main.yml) [![goreportcard](https://goreportcard.com/badge/github.com/formancehq/payments)](https://goreportcard.com/report/github.com/formancehq/payments) [![codecov](https://codecov.io/github/formancehq/payments/graph/badge.svg?token=SrhCCbrtnV)](https://codecov.io/github/formancehq/payments)

# Getting started

Payments works as a standalone binary, the latest of which can be downloaded from the [releases page](https://github.com/formancehq/payments/releases).
You can move the binary to any executable path, such as to `/usr/local/bin`. Installing it locally using Docker is also
possible, and probably easier for testing purposes as it comes prepackaged with all the dependencies.

Note that you need to set up the `STACK_PUBLIC_URL` env variable to set up a publicly available URL so that webhooks
and redirects from the Payment Service Providers (PSP) can be sent to the application.
You can use [NGrok](https://ngrok.com/) for that, for example. If you don't plan on using any connectors with webhooks,
you could simply set it as localhost (or anything, really).

```SHELL
$ git clone git@github.com:formancehq/payments.git
$ cd payments
$ just compile-plugins
$ STACK_PUBLIC_URL=http://localhost docker compose up
```

## Debugging
You can also use the docker-compose.dev.yml file to run the application with Delve and Air, which allow debugging and
live reloading.

## Use console as a frontend

The payment application comes with a console frontend when deploying through docker-compose (with or without debugging).
You can access it at http://localhost:3000/formance/localhost?region=localhost.

# What is it?

Basically, a framework.

A framework to ingest payin and payout coming from different payment providers (PSP).

The framework contains connectors. Each connector is basically a translator for a PSP.
Translator, because the main role of a connector is to translate specific PSP payin/payout formats to a generalized format used at Formance.

Because it is a framework, it is extensible. Please follow the guide below if you want to add your connector.

# Contribute

Please see the following documents:
- Connector development tutorial: [CONTRIBUTING.md](./CONTRIBUTING.md)
- General development guidelines: [CONTRIBUTING_GUIDE.md](./CONTRIBUTING_GUIDE.md)
