FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o filesystem-gateway ./cmd/server

# ----

FROM alpine:3.21

RUN apk add --no-cache ca-certificates zstd

WORKDIR /app
COPY --from=builder /build/filesystem-gateway .

# Runs as root — this service manages files on behalf of other workloads
# (mkdir, chown, extract archives). Security is enforced via bearer auth,
# path traversal prevention, and NFS mount scope.
EXPOSE 8080

ENTRYPOINT ["/app/filesystem-gateway"]
