# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go module files
COPY go.mod go.sum go.work* ./

# Copy all library modules
COPY libs/ ./libs/

# Copy internal modules
COPY internal/ ./internal/

# Copy tools module required by go.work
COPY tools/ ./tools/

# Copy the research entrypoint
COPY cmd/research/ ./cmd/research/

RUN go mod download

ARG VERSION=0.1.0
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /research \
    ./cmd/research

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget
RUN addgroup -S jax && adduser -S jax -G jax

WORKDIR /app
COPY --from=builder /research /app/research

RUN chown jax:jax /app/research
USER jax

EXPOSE 8091

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD wget -q --spider http://localhost:8091/health || exit 1

ENTRYPOINT ["/app/research"]
