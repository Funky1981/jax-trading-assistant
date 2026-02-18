FROM golang:1.25 AS build

WORKDIR /src

# Copy module files and local libs first for better caching
COPY go.mod go.sum ./
COPY libs/ ./libs/
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/jax-memory ./services/jax-memory/cmd/jax-memory

FROM alpine:3.20
RUN apk add --no-cache ca-certificates

COPY --from=build /out/jax-memory /usr/local/bin/jax-memory

ENV PORT=8090
EXPOSE 8090

ENTRYPOINT ["/usr/local/bin/jax-memory"]
