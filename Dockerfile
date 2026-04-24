# ─────────────────────────────────────────────────────────────────────────────
#  ZATRANO — Enterprise Multi-stage Dockerfile
#  Base: Go 1.26.1 (Fiber v3 requires ≥ 1.23)
#  Final: distroless/static — minimal saldırı yüzeyi, shell yok
# ─────────────────────────────────────────────────────────────────────────────

# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.26.1-alpine AS builder

# Güvenlik: minimal Alpine + gerekli araçlar
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Bağımlılıkları önce kopyala (Docker layer cache)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Kaynak kodları kopyala
COPY . .

# Production binary — version build-time'da inject edilir
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -extldflags '-static' -X main.version=${VERSION}" \
    -trimpath \
    -o /app/bin/zatrano \
    ./main.go

# ── Stage 2: Runner ───────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS runner

WORKDIR /app

# CA sertifikaları (HTTPS çağrıları için)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Timezone verisi
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Binary
COPY --from=builder /app/bin/zatrano .

# Statik varlıklar
COPY --from=builder /app/views ./views
COPY --from=builder /app/public ./public

# Yükleme dizini için volume
VOLUME ["/app/uploads"]

# Health check — /health endpoint'i kontrol eder
HEALTHCHECK \
  --interval=30s \
  --timeout=5s \
  --start-period=15s \
  --retries=3 \
  CMD ["/app/zatrano", "--version"] || exit 1

EXPOSE 3000

USER nonroot:nonroot

ENTRYPOINT ["/app/zatrano"]
