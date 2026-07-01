set dotenv-load

default:
  @just --list

pre-commit: tidy generate lint openapi compile-plugins compile-connector-capabilities
pc: pre-commit

lint:
  @golangci-lint run --fix --build-tags it --timeout 5m

tidy:
  @go mod tidy
  @cd pkg/domain && go mod tidy

compile-plugins:
  ./tools/compile-plugins/compile-plugin.sh

[group('openapi')]
validate-openapi:
  @go run github.com/getkin/kin-openapi/cmd/validate@v0.135.0 openapi.yaml

[group('openapi')]
compile-connector-configs:
    @go build -o compile-configs {{justfile_directory()}}/tools/compile-configs
    ./compile-configs --path {{justfile_directory()}}/internal/connectors/plugins/public --path {{justfile_directory()}}/ee/plugins --output {{justfile_directory()}}/openapi/v3/v3-connectors-config.yaml
    @rm ./compile-configs

compile-connector-capabilities:
    @go build -tags ee -o compile-capabilities {{justfile_directory()}}/tools/compile-capabilities
    ./compile-capabilities --path {{justfile_directory()}}/internal/connectors/plugins/public --path {{justfile_directory()}}/ee/plugins --output {{justfile_directory()}}/docs/other/connector-capabilities.json
    @rm ./compile-capabilities

[group('openapi')]
compile-api-yaml: compile-connector-configs
    @npx openapi-merge-cli --config {{justfile_directory()}}/openapi/openapi-merge.json
    @yq -oy {{justfile_directory()}}/openapi.json > openapi.yaml
    @rm {{justfile_directory()}}/openapi.json

[group('openapi')]
compile-api-docs:
    @npx openapi-merge-cli --config {{justfile_directory()}}/openapi/openapi-docs-merge.json
    @npx -y widdershins {{justfile_directory()}}/openapi.json -o {{justfile_directory()}}/docs/api/README.md --search false --language_tabs 'http:HTTP' --summary --omitHeader
    @rm {{justfile_directory()}}/openapi.json

[group('openapi')]
openapi: compile-api-yaml compile-api-docs validate-openapi

[group('test')]
tests:
  @go test -race -covermode=atomic \
    -coverprofile coverage.txt \
    -tags it \
    ./...
  @cd pkg/domain && go test -race ./...

# Contract tests call real connector sandbox APIs to detect upstream API drift.
# Gated behind the `contract` build tag so they never run as part of `tests`.
# Requires the connector's contract credentials in the environment, e.g. for
# adyen: ADYEN_CONTRACT_API_KEY and ADYEN_CONTRACT_COMPANY_ID. Without them the
# suite skips rather than fails. Run daily via .github/workflows/contract-tests.yml.
# Scoped to the `client` package (where contract tests live) and run with -v so
# bootstrap logs (e.g. paste-ready pinned-ID literals) are visible: `go test`
# discards a passing package's stdout/stderr unless -v is set.
[group('test')]
contract-tests connector="adyen":
  @go test -tags contract -v -count=1 ./internal/connectors/plugins/public/{{connector}}/client/...  -args -ginkgo.v

[group('test')]
generate-sdk: openapi
    @export PATH=$PATH:$(go env GOPATH)/bin && cd pkg/client && speakeasy run --skip-versioning

[group('test')]
generate: generate-sdk
    @go generate ./...
    @cd pkg/domain && go generate ./...

[group('build')]
build-ce: compile-plugins
    go build -ldflags "-X github.com/formancehq/payments/cmd.Edition=community" -o payments .

[group('build')]
build-ee: compile-plugins
    go build -tags ee -ldflags "-X github.com/formancehq/payments/cmd.Edition=enterprise" -o payments-ee .

[group('releases')]
release-local:
    @goreleaser release --nightly --skip=publish --clean

[group('releases')]
release-ci:
    @goreleaser release --nightly --clean --parallelism 2

[group('releases')]
release:
    @goreleaser release --clean --parallelism 2

[group('deploy')]
deploy server auth-token application additional-args:
    @argocd app set --auth-token {{auth-token}} --server {{server}} {{application}} --grpc-web {{additional-args}}
    @argocd app sync --auth-token {{auth-token}} --server {{server}} {{application}} --grpc-web

[group('plugins')]
bootstrap-plugin CONNECTOR_NAME EDITION="public":
    @go build -o connector-template {{justfile_directory()}}/tools/connector-template
    @if [ "{{EDITION}}" = "enterprise" ]; then \
        ./connector-template --connector-dir-path {{justfile_directory()}}/ee/plugins --connector-name {{CONNECTOR_NAME}}; \
    else \
        ./connector-template --connector-dir-path {{justfile_directory()}}/internal/connectors/plugins/public --connector-name {{CONNECTOR_NAME}}; \
    fi
    @rm ./connector-template
