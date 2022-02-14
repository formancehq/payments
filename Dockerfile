FROM golang:1.17-buster as src
COPY . /app
WORKDIR /app

FROM src as dev
RUN apt-get update && apt-get install -y ca-certificates git-core ssh
RUN git config --global url.ssh://git@github.com/numary.insteadOf https://github.com/numary

FROM src as compiler
ARG VERSION=latest
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X github.com/numary/payment/cmd.Version=${VERSION}" -o payments-cloud .

FROM alpine as app
RUN apk add --no-cache ca-certificates curl
COPY --from=compiler /app/payments-cloud /usr/local/bin/payments
EXPOSE 8080
CMD ["payments"]
