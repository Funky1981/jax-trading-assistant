# Build stage
FROM golang:1.25.4-alpine@sha256:d3f0cf7723f3429e3f9ed846243970b20a2de7bae6a5b66fc5914e228d831bbb AS builder

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
FROM alpine:3.19@sha256:6baf43584bcb78f2e5847d1de515f23499913ac9f12bdf834811a3145eb11ca1

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -S jax && adduser -S jax -G jax

WORKDIR /app

# Copy binary from builder
COPY --from=builder /trader /app/trader
COPY config/ /app/config/

# Include Go toolchain + module cache + source for in-container trust-gate tests.
COPY --from=builder /usr/local/go /usr/local/go
COPY --from=builder /go/pkg/mod /go/pkg/mod
COPY go.mod go.sum go.work* /app/src/
COPY libs/ /app/src/libs/
COPY internal/ /app/src/internal/
COPY tools/ /app/src/tools/
COPY cmd/ /app/src/cmd/
COPY tests/ /app/src/tests/

ENV GOPATH=/go
ENV PATH="/usr/local/go/bin:${PATH}"

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
