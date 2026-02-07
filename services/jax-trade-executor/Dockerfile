FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
COPY libs/ ./libs/
COPY services/jax-trade-executor/ ./services/jax-trade-executor/

# Download dependencies
RUN cd services/jax-trade-executor && go mod download

# Build
RUN cd services/jax-trade-executor && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/jax-trade-executor ./cmd/jax-trade-executor

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /app/jax-trade-executor .

EXPOSE 8097

CMD ["./jax-trade-executor"]
