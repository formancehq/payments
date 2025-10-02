# syntax=docker/dockerfile:1

# Development Dockerfile for payments service
# Based on ebics-bridge structure but optimized for development workflow

FROM golang:1.24.7-alpine as development

# Install minimal dependencies including build tools
RUN apk add --no-cache \
    git \
    curl \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    make

# Install delve (debugger) and air (hot reload)
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go install github.com/air-verse/air@v1.61.7

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Create minimal structure for local replacements
RUN mkdir -p pkg/client internal/connectors/plugins/public/generic/client/generated

# Copy only the go.mod files for local replacements
COPY pkg/client/go.mod pkg/client/
COPY internal/connectors/plugins/public/generic/client/generated/go.mod internal/connectors/plugins/public/generic/client/generated/

# Download dependencies (this will be cached in Docker layer)
RUN go mod download

# Copy source code
COPY . .

# Set environment variables for development
ENV GO111MODULE=on
ENV CGO_ENABLED=1
ENV DEBUG=true

# Expose ports
EXPOSE 8080 9090 2345

# No command, it needs to be set by docker-compose (different use cases)