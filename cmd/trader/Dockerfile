# Build stage
FROM golang:1.25-alpine AS builder

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
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -S jax && adduser -S jax -G jax

WORKDIR /app

# Copy binary from builder
COPY --from=builder /trader /app/trader

# Change ownership
RUN chown -R jax:jax /app

# Switch to non-root user
USER jax

# Expose port
EXPOSE 8100

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget -q --spider http://localhost:8081/health || exit 1

# Run the application
ENTRYPOINT ["/app/trader"]
