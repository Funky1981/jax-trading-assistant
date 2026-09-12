FROM golang:1.25.4-alpine@sha256:d3f0cf7723f3429e3f9ed846243970b20a2de7bae6a5b66fc5914e228d831bbb AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum go.work* ./
COPY libs/ ./libs/
COPY tools/ ./tools/
COPY cmd/db-migrate/ ./cmd/db-migrate/

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /db-migrate ./cmd/db-migrate

FROM alpine:3.19@sha256:6baf43584bcb78f2e5847d1de515f23499913ac9f12bdf834811a3145eb11ca1

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /db-migrate /app/db-migrate

ENTRYPOINT ["/app/db-migrate"]
