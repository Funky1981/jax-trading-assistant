FROM golang:1.26.8-alpine@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628 AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum go.work* ./
COPY libs/ ./libs/
COPY tools/ ./tools/
COPY cmd/db-migrate/ ./cmd/db-migrate/

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /db-migrate ./cmd/db-migrate

FROM alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40

RUN apk add --no-cache ca-certificates tzdata
RUN apk upgrade --no-cache

WORKDIR /app
COPY --from=builder /db-migrate /app/db-migrate

ENTRYPOINT ["/app/db-migrate"]
