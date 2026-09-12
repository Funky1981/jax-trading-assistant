# Build stage
FROM golang:1.26.8-alpine@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628 AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go module files
COPY go.mod go.sum go.work* ./

# Copy all library modules
COPY libs/ ./libs/

# Copy internal modules
COPY internal/ ./internal/

# Copy tools module (required by go.work)
COPY tools/ ./tools/

# Copy cmd/trader
COPY cmd/trader/ ./cmd/trader/

# Download dependencies
RUN go mod download

# Build the application
ARG VERSION=0.1.0
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /trader \
    ./cmd/trader

# Runtime stage
FROM alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata
RUN apk upgrade --no-cache

# Create non-root user
RUN addgroup -S jax && adduser -S jax -G jax

WORKDIR /app

# Copy binary from builder
COPY --from=builder /trader /app/trader
COPY config/ /app/config/

# The commercial runtime contains only the application and its configuration.
# Trust-gate source execution remains a development/test concern and is
# intentionally not bundled with the shipped trader image.

# Change ownership
RUN chown -R jax:jax /app

# Switch to non-root user
USER jax

# Expose port
EXPOSE 8100

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget -q --spider http://localhost:8100/ready || exit 1

# Run the application
ENTRYPOINT ["/app/trader"]
