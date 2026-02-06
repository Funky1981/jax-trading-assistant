# Build stage
FROM golang:1.24 AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
COPY libs/ ./libs/
RUN go mod download

# Copy source
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/jax-signal-generator ./services/jax-signal-generator/cmd/jax-signal-generator

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /bin/jax-signal-generator /app/jax-signal-generator
COPY config/jax-signal-generator.json /app/config/jax-signal-generator.json

EXPOSE 8096

CMD ["/app/jax-signal-generator", "-config", "/app/config/jax-signal-generator.json"]
