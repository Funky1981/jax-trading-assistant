FROM golang:1.24 AS build

WORKDIR /src

# Copy module files and local libs first for better caching
COPY go.mod go.sum ./
COPY libs/ ./libs/
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/jax-api ./services/jax-api/cmd/jax-api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates

COPY --from=build /out/jax-api /usr/local/bin/jax-api

# Copy config files
COPY config /config

ENV PORT=8081
EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/jax-api", "-providers", "/config/providers.json", "-config", "/config/jax-core.json", "-strategies", "/config/strategies"]
