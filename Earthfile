VERSION 0.8
PROJECT FormanceHQ/payments

IMPORT github.com/formancehq/earthly:tags/v0.16.2 AS core

FROM core+base-image

sources:
    WORKDIR src
    WORKDIR /src
    COPY go.* .
    COPY --dir pkg cmd internal .
    COPY main.go .
    SAVE ARTIFACT /src

compile:
    FROM core+builder-image
    COPY (+sources/*) /src
    WORKDIR /src
    ARG VERSION=latest
    DO --pass-args core+GO_COMPILE --VERSION=$VERSION

build-image:
    FROM core+final-image
    ENTRYPOINT ["/bin/payments"]
    CMD ["serve"]
    COPY (+compile/main) /bin/payments
    ARG REPOSITORY=ghcr.io
    ARG tag=latest
    DO core+SAVE_IMAGE --COMPONENT=payments --REPOSITORY=${REPOSITORY} --TAG=$tag

tests:
    FROM core+builder-image
    COPY (+sources/*) /src
    WORKDIR /src
    WITH DOCKER --pull=postgres:15-alpine
        DO --pass-args core+GO_TESTS
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
    SAVE ARTIFACT cmd AS LOCAL cmd
    SAVE ARTIFACT internal AS LOCAL internal
    SAVE ARTIFACT pkg AS LOCAL pkg
    SAVE ARTIFACT main.go AS LOCAL main.go

pre-commit:
    WAIT
      BUILD --pass-args +tidy
    END
    BUILD --pass-args +lint

openapi:
    COPY ./openapi.yaml .
    SAVE ARTIFACT ./openapi.yaml

tidy:
    FROM core+builder-image
    COPY --pass-args (+sources/src) /src
    WORKDIR /src
    DO --pass-args core+GO_TIDY

generate-generic-connector-client:
    FROM openapitools/openapi-generator-cli:v6.6.0
    WORKDIR /src
    COPY cmd/connectors/internal/connectors/generic/client/generic-openapi.yaml .
    RUN docker-entrypoint.sh generate \
        -i ./generic-openapi.yaml \
        -g go \
        -o ./generated \
        --git-user-id=formancehq \
        --git-repo-id=payments \
        -p packageVersion=latest \
        -p isGoSubmodule=true \
        -p packageName=genericclient
    RUN rm -rf ./generated/test
    SAVE ARTIFACT ./generated AS LOCAL ./cmd/connectors/internal/connectors/generic/client/generated

release:
    FROM core+builder-image
    ARG mode=local
    COPY --dir . /src
    DO core+GORELEASER --mode=$mode