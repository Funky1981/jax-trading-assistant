FROM golang:1.24 AS build

WORKDIR /workspace

# Copy root go files for workspace context
COPY go.mod go.sum ./

# Copy all libs needed by orchestrator
COPY libs/ ./libs/

# Copy service files
COPY services/jax-orchestrator/ ./services/jax-orchestrator/

# Change to service directory and build
WORKDIR /workspace/services/jax-orchestrator
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/jax-orchestrator-http ./cmd/jax-orchestrator-http

FROM alpine:3.20
RUN apk add --no-cache ca-certificates

COPY --from=build /out/jax-orchestrator-http /usr/local/bin/jax-orchestrator-http

ENV PORT=8091
EXPOSE 8091

CMD ["/usr/local/bin/jax-orchestrator-http"]

ENV PORT=8091
EXPOSE 8091

ENTRYPOINT ["/usr/local/bin/jax-orchestrator-http", "-port", "8091"]
