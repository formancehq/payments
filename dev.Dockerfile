# syntax=docker/dockerfile:1

# Development Dockerfile for payments service
# Based on ebics-bridge structure but optimized for development workflow

FROM golang:1.26-alpine as development

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
RUN CGO_ENABLED=0 go install github.com/go-delve/delve/cmd/dlv@v1.26
RUN CGO_ENABLED=0 go install github.com/air-verse/air@v1.61.7

# Create a custom user with appropriate permissions
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Set proper ownership and permissions for the working directory
RUN chown -R appuser:appgroup /app

# Create tmp directory and set permissions
RUN mkdir -p /app/tmp && chown -R appuser:appgroup /app/tmp && chmod 755 /app/tmp

# Create Go cache directory for appuser
RUN mkdir -p /home/appuser/.cache/go-build && chown -R appuser:appgroup /home/appuser/.cache

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Copy go.mod/go.sum for all local replacements so go mod download can resolve them
COPY --parents pkg/client/go.mod pkg/domain/go.mod pkg/domain/go.sum ./
COPY --parents ce/plugins/*/go.mod ce/plugins/*/go.sum ./

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
# Switch to the custom user
USER appuser

# No command, it needs to be set by docker-compose (different use cases)
