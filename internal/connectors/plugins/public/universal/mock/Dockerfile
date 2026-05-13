FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/mock ./internal/connectors/plugins/public/universal/mock

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mock /mock
EXPOSE 8080
ENTRYPOINT ["/mock"]
