FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum go.work* ./
COPY libs/ ./libs/
COPY tools/ ./tools/
COPY cmd/db-migrate/ ./cmd/db-migrate/

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /db-migrate ./cmd/db-migrate

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /db-migrate /app/db-migrate

ENTRYPOINT ["/app/db-migrate"]
