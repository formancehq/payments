set dotenv-load

default:
  @just --list

pre-commit: generate tidy lint
pc: pre-commit

lint:
  @golangci-lint run --fix --build-tags it --timeout 5m

tidy:
  @go mod tidy

[group('openapi')]
compile-connector-configs:
    @go build -o compile-configs {{justfile_directory()}}/tools/compile-configs
    ./compile-configs --path {{justfile_directory()}}/internal/connectors/plugins/public --output {{justfile_directory()}}/openapi/v3/v3-connectors-config.yaml
    @rm ./compile-configs

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
openapi: compile-api-yaml compile-api-docs

[group('test')]
tests:
  @go test -race -covermode=atomic \
    -coverprofile coverage.txt \
    -tags it \
    ./...

[group('test')]
generate-sdk: openapi
    @export PATH=$PATH:$(go env GOPATH)/bin && cd pkg/client && speakeasy run --skip-versioning

[group('test')]
generate: generate-sdk
    @go generate ./...

[group('releases')]
release-local:
    @goreleaser release --nightly --skip=publish --clean

[group('releases')]
release-ci:
    @goreleaser release --nightly --clean

[group('releases')]
release:
    @goreleaser release --clean
