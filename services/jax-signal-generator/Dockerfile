# Build stage
FROM golang:1.24 AS builder

WORKDIR /workspace

# Copy go mod files from root
COPY go.mod go.sum ./
COPY libs/ ./libs/
COPY services/jax-signal-generator/ ./services/jax-signal-generator/

WORKDIR /workspace/services/jax-signal-generator
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/jax-signal-generator ./cmd/jax-signal-generator

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /bin/jax-signal-generator /app/jax-signal-generator
COPY config/jax-signal-generator.json /app/config/jax-signal-generator.json

EXPOSE 8096

CMD ["/app/jax-signal-generator", "-config", "/app/config/jax-signal-generator.json"]
