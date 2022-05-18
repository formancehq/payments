FROM --platform=$BUILDPLATFORM golang:1.18 AS builder
RUN apt-get update && \
    apt-get install -y gcc-aarch64-linux-gnu gcc-x86-64-linux-gnu && \
    ln -s /usr/bin/aarch64-linux-gnu-gcc /usr/bin/arm64-linux-gnu-gcc  && \
    ln -s /usr/bin/x86_64-linux-gnu-gcc /usr/bin/amd64-linux-gnu-gcc
# 1. Precompile the entire go standard library into the first Docker cache layer: useful for other projects too!
RUN CGO_ENABLED=1 GOOS=linux go install -v -installsuffix cgo -a std
ARG TARGETARCH
ARG APP_SHA
ARG VERSION
WORKDIR /go/src/github.com/numary/payments
# get deps first so it's cached
COPY go.mod .
COPY go.sum .
RUN --mount=type=cache,id=gomod,target=/go/bridge/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go mod download
COPY . .
RUN --mount=type=cache,id=gomod,target=/go/bridge/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH \
    CC=$TARGETARCH-linux-gnu-gcc \
    go build -o payments \
    -ldflags="-X github.com/numary/payments/cmd.Version=${VERSION} \
    -X github.com/numary/payments/cmd.BuildDate=$(date +%s) \
    -X github.com/numary/payments/cmd.Commit=${APP_SHA}" ./

FROM ubuntu:jammy
RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /go/src/github.com/numary/payments/payments /usr/local/bin/payments
EXPOSE 8080
CMD ["payments"]
