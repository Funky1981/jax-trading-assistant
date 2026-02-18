FROM golang:1.25 AS builder

WORKDIR /workspace

# Copy go module files and lib dependencies
COPY go.mod go.sum ./
COPY libs/ ./libs/

RUN go mod download

# Copy all source code
COPY . .

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/jax-market ./services/jax-market/cmd/jax-market

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /bin/jax-market /app/jax-market

# Copy config
COPY config/jax-market.json /app/config/jax-market.json

EXPOSE 8095

CMD ["/app/jax-market", "-config", "/app/config/jax-market.json"]
