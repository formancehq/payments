FROM golang:1.19.3-bullseye AS builder

RUN apt-get update && \
    apt-get install -y gcc-aarch64-linux-gnu gcc-x86-64-linux-gnu && \
    ln -s /usr/bin/aarch64-linux-gnu-gcc /usr/bin/arm64-linux-gnu-gcc  && \
    ln -s /usr/bin/x86_64-linux-gnu-gcc /usr/bin/amd64-linux-gnu-gcc

# 1. Precompile the entire go standard library into the first Docker cache layer: useful for other projects too!
RUN CGO_ENABLED=0 GOOS=linux go install -v -installsuffix cgo -a std

ARG TARGETARCH
ARG APP_SHA
ARG VERSION

WORKDIR /go/src/github.com/formancehq/payments

# get deps first so it's cached
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    CC=$TARGETARCH-linux-gnu-gcc \
    go build -o bin/payments \
    -ldflags="-X github.com/formancehq/payments/cmd.Version=${VERSION} \
    -X github.com/formancehq/payments/cmd.BuildDate=$(date +%s) \
    -X github.com/formancehq/payments/cmd.Commit=${APP_SHA}" ./

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/formancehq/payments/bin/payments /usr/local/bin/payments

EXPOSE 8080

ENTRYPOINT ["payments"]

CMD ["server"]
