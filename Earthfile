VERSION 0.8
PROJECT FormanceHQ/payments

IMPORT github.com/formancehq/earthly:tags/v0.19.1 AS core

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

release:
    FROM core+builder-image
    ARG mode=local
    COPY --dir . /src
    COPY (+compile-plugins/list.go) /src/internal/connectors/plugins/public/list.go
    DO core+GORELEASER --mode=$mode

openapi:
    COPY openapi.yaml /openapi.yaml
    SAVE ARTIFACT /openapi.yaml AS LOCAL openapi.yaml