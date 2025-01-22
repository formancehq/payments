VERSION 0.8
PROJECT FormanceHQ/payments

IMPORT github.com/formancehq/earthly:tags/v0.19.0 AS core

FROM core+base-image

postgres:
    FROM postgres:15-alpine

sources:
    WORKDIR src
    WORKDIR /src
    COPY go.* .
    COPY --dir cmd pkg internal tools .
    COPY main.go .
    SAVE ARTIFACT /src

compile-plugins:
    FROM core+builder-image
    COPY (+sources/*) /src
    WORKDIR /src/internal/connectors/plugins/public
    RUN printf "package public\n\n" > list.go
    RUN printf "import (\n" >> list.go
    FOR c IN $(ls -d */ | sed 's#/##')
        RUN printf "    _ \"github.com/formancehq/payments/internal/connectors/plugins/public/$c\"\n" >> list.go
    END
    RUN printf ")\n" >> list.go
    SAVE ARTIFACT /src/internal/connectors/plugins/public/list.go /list.go

compile:
    FROM core+builder-image
    COPY (+sources/*) /src
    COPY (+compile-plugins/list.go) /src/internal/connectors/plugins/public/list.go
    WORKDIR /src
    ARG VERSION=latest
    DO --pass-args core+GO_COMPILE --VERSION=$VERSION

build-image:
    FROM core+final-image
    ENTRYPOINT ["/bin/payments"]
    CMD ["serve"]
    COPY (+compile/main) /bin/payments
    FOR c IN $(ls /plugins/*)
        RUN chmod +x $c
    END
    ARG REPOSITORY=ghcr.io
    ARG tag=latest
    DO core+SAVE_IMAGE --COMPONENT=payments --REPOSITORY=${REPOSITORY} --TAG=$tag

tests:
    FROM +tidy
    COPY (+sources/*) /src
    WORKDIR /src

    ARG includeIntegrationTests="true"
    ARG coverage=""

    ENV CGO_ENABLED=1 # required for -race

    LET goFlags="-race"

    IF [ "$coverage" = "true" ]
        SET goFlags="$goFlags -covermode=atomic"
        SET goFlags="$goFlags -coverpkg=./..."
        SET goFlags="$goFlags -coverprofile coverage.txt"
    END

    IF [ "$includeIntegrationTests" = "true" ]
        COPY (+compile-plugins/list.go) /src/internal/connectors/plugins/public/list.go
        SET goFlags="$goFlags -tags it"
        WITH DOCKER --load=postgres:15-alpine=+postgres
            RUN go test $goFlags ./...
        END
    ELSE
        WITH DOCKER --pull=postgres:15-alpine
            DO --pass-args +GO_TESTS
        END
    END

    IF [ "$coverage" = "true" ]
        SAVE ARTIFACT coverage.txt AS LOCAL coverage.txt
    END

deploy:
    COPY (+sources/*) /src
    LET tag=$(tar cf - /src | sha1sum | awk '{print $1}')
    WAIT
        BUILD --pass-args +build-image --tag=$tag
    END
    FROM --pass-args core+vcluster-deployer-image
    RUN kubectl patch Versions.formance.com default -p "{\"spec\":{\"payments\": \"${tag}\"}}" --type=merge

deploy-staging:
    BUILD --pass-args core+deploy-staging

lint:
    FROM core+builder-image
    COPY (+sources/*) /src
    COPY --pass-args +tidy/go.* .
    WORKDIR /src
    DO --pass-args core+GO_LINT
    COPY (+compile-plugins/list.go) .
    SAVE ARTIFACT cmd AS LOCAL cmd
    SAVE ARTIFACT internal AS LOCAL internal
    SAVE ARTIFACT pkg AS LOCAL pkg
    SAVE ARTIFACT main.go AS LOCAL main.go
    SAVE ARTIFACT list.go AS LOCAL internal/connectors/plugins/public/list.go

pre-commit:
    WAIT
      BUILD --pass-args +tidy
    END
    BUILD --pass-args +lint

compile-openapi-configs:
    FROM core+builder-image
    COPY (+sources/*) /src
    WORKDIR /src/tools/compile-configs
    RUN go build -o compile-configs
    RUN ./compile-configs --path /src/internal/connectors/plugins/public --output ./v3-connectors-config.yaml
    SAVE ARTIFACT ./v3-connectors-config.yaml /v3-connectors-config.yaml

openapi:
    FROM node:20-alpine
    RUN apk update && apk add yq
    RUN npm install -g openapi-merge-cli
    WORKDIR /src
    COPY --dir openapi openapi
    COPY (+compile-openapi-configs/v3-connectors-config.yaml) ./openapi/v3/v3-connectors-config.yaml
    RUN openapi-merge-cli --config ./openapi/openapi-merge.json
    RUN yq -oy ./openapi.json > openapi.yaml
    SAVE ARTIFACT ./openapi.yaml AS LOCAL ./openapi.yaml

tidy:
    FROM core+builder-image
    COPY --pass-args (+sources/src) /src
    WORKDIR /src
    COPY --dir test .
    DO --pass-args core+GO_TIDY

generate:
    FROM core+builder-image
    RUN apk update && apk add openjdk11
    DO --pass-args core+GO_INSTALL --package=go.uber.org/mock/mockgen@latest
    COPY (+sources/*) /src
    WORKDIR /src
    DO --pass-args core+GO_GENERATE
    SAVE ARTIFACT internal AS LOCAL internal

# generate-generic-connector-client:
#     FROM openapitools/openapi-generator-cli:v6.6.0
#     WORKDIR /src
#     COPY cmd/connectors/internal/connectors/generic/client/generic-openapi.yaml .
#     RUN docker-entrypoint.sh generate \
#         -i ./generic-openapi.yaml \
#         -g go \
#         -o ./generated \
#         --git-user-id=formancehq \
#         --git-repo-id=payments \
#         -p packageVersion=latest \
#         -p isGoSubmodule=true \
#         -p packageName=genericclient
#     RUN rm -rf ./generated/test
#     SAVE ARTIFACT ./generated AS LOCAL ./cmd/connectors/internal/connectors/generic/client/generated

release:
    FROM core+builder-image
    ARG mode=local
    COPY --dir . /src
    COPY (+compile-plugins/list.go) /src/internal/connectors/plugins/public/list.go
    DO core+GORELEASER --mode=$mode
