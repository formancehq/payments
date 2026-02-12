VERSION 0.8
PROJECT FormanceHQ/payments

ARG core=github.com/formancehq/earthly:main
IMPORT $core AS core

FROM core+base-image

postgres:
    FROM postgres:15-alpine

sources:
    WORKDIR src
    WORKDIR /src
    COPY go.* .
    COPY --dir cmd pkg internal tools ee .
    COPY main.go .
    COPY main_ee.go .
    SAVE ARTIFACT /src

compile-plugins:
    ARG local_save=false
    FROM core+builder-image
    COPY (+sources/*) /src
    RUN ./src/tools/compile-plugins/compile-plugin.sh list.go /src/internal/connectors/plugins/public public github.com/formancehq/payments/internal/connectors/plugins/public
    RUN ./src/tools/compile-plugins/compile-plugin.sh list.go /src/ee/plugins plugins github.com/formancehq/payments/ee/plugins
    SAVE ARTIFACT /src/internal/connectors/plugins/public/list.go /public-list.go
    SAVE ARTIFACT /src/ee/plugins/list.go /enterprise-list.go
    IF $local_save
        SAVE ARTIFACT /src/internal/connectors/plugins/public/list.go AS LOCAL internal/connectors/plugins/public/list.go
        SAVE ARTIFACT /src/ee/plugins/list.go AS LOCAL ee/plugins/list.go
    END

compile:
    FROM core+builder-image
    COPY (+sources/*) /src
    COPY (+compile-plugins/public-list.go) /src/internal/connectors/plugins/public/list.go
    COPY (+compile-plugins/enterprise-list.go) /src/ee/plugins/list.go
    WORKDIR /src
    ARG VERSION=latest
    DO --pass-args core+GO_COMPILE --VERSION=$VERSION --ADDITIONAL_ARGUMENTS="-tags ee"

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
    COPY (+compile-plugins/public-list.go) /src/internal/connectors/plugins/public/list.go
    COPY (+compile-plugins/enterprise-list.go) /src/ee/plugins/list.go
    DO core+GORELEASER --mode=$mode

openapi:
    COPY openapi.yaml /openapi.yaml
    SAVE ARTIFACT /openapi.yaml AS LOCAL openapi.yaml
