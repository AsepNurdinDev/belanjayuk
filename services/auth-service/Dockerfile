# =============================================================
# Stage 1: Builder
# =============================================================
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o auth-service \
    ./cmd/api

# =============================================================
# Stage 2: Runtime
# =============================================================
FROM scratch

# Copy timezone data dan CA certs dari builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /build/auth-service /auth-service

# Non-root user (UID 65534 = nobody)
USER 65534:65534

# Expose ports
EXPOSE 8001 9001

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD ["/auth-service", "-health"] || exit 1

ENTRYPOINT ["/auth-service"]
