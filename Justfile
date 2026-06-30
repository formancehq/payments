set dotenv-load

default:
  @just --list

pre-commit: tidy generate lint openapi compile-plugins compile-connector-capabilities
pc: pre-commit

lint:
  @golangci-lint run --fix --build-tags it --timeout 5m
  @set -e; for d in ce/plugins/*/; do cd "{{justfile_directory()}}/$d" && golangci-lint run --fix --build-tags it --timeout 5m && cd "{{justfile_directory()}}"; done

tidy:
  @go run {{justfile_directory()}}/tools/sync-ce-plugins --connector-dir-path {{justfile_directory()}}/ce/plugins
  @go mod tidy
  @cd pkg/domain && go mod tidy
  @set -e; for d in ce/plugins/*/; do cd "{{justfile_directory()}}/$d" && go mod tidy && cd "{{justfile_directory()}}"; done

compile-plugins:
  ./tools/compile-plugins/compile-plugin.sh

[group('openapi')]
validate-openapi:
  @go run github.com/getkin/kin-openapi/cmd/validate@v0.135.0 openapi.yaml

[group('openapi')]
compile-connector-configs:
    @go build -o compile-configs {{justfile_directory()}}/tools/compile-configs
    ./compile-configs --path {{justfile_directory()}}/ce/plugins --path {{justfile_directory()}}/ee/plugins --output {{justfile_directory()}}/openapi/v3/v3-connectors-config.yaml
    @rm ./compile-configs

compile-connector-capabilities:
    @go build -tags ee -o compile-capabilities {{justfile_directory()}}/tools/compile-capabilities
    ./compile-capabilities --path {{justfile_directory()}}/ce/plugins --path {{justfile_directory()}}/ee/plugins --output {{justfile_directory()}}/docs/other/connector-capabilities.json
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
  @set -e; for d in ce/plugins/*/; do \
    name=$(basename "$d"); \
    cd "{{justfile_directory()}}/$d" && \
    go test -race -covermode=atomic -coverprofile "{{justfile_directory()}}/coverage-plugin-$name.txt" -tags it ./... && \
    cd "{{justfile_directory()}}"; \
  done
  @for f in coverage-plugin-*.txt; do tail -n +2 "$f" >> coverage.txt && rm "$f"; done

# Contract tests call real connector sandbox APIs to detect upstream API drift.
# Gated behind the `contract` build tag so they never run as part of `tests`.
# Requires the connector's contract credentials in the environment, e.g. for
# adyen: ADYEN_CONTRACT_API_KEY and ADYEN_CONTRACT_COMPANY_ID. Without them the
# suite skips rather than fails. Run daily via .github/workflows/contract-tests.yml.
[group('test')]
contract-tests connector="adyen":
  @dir="ce/plugins/{{connector}}"; \
  if [ -d "ee/plugins/{{connector}}" ]; then dir="ee/plugins/{{connector}}"; fi; \
  cd "$dir" && go test -tags contract -count=1 ./...

[group('test')]
generate-sdk: openapi
    @export PATH=$PATH:$(go env GOPATH)/bin && cd pkg/client && speakeasy run --skip-versioning

[group('test')]
generate: generate-sdk
    @go generate ./...
    @cd pkg/domain && go generate ./...
    @set -e; for d in ce/plugins/*/; do cd "{{justfile_directory()}}/$d" && go generate ./... && cd "{{justfile_directory()}}"; done

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
        ./connector-template --connector-dir-path {{justfile_directory()}}/ce/plugins --connector-name {{CONNECTOR_NAME}}; \
    fi
    @rm ./connector-template
